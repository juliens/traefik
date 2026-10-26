package wasihttp

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"os"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/sys"
)

const wasiModuleName = "wasi_snapshot_preview1"

const (
	fdflagNonblock = 0x0004
	filetypeChar   = 2
	filetypeStream = 6

	eventClock   = 0
	eventFDRead  = 1
	eventFDWrite = 2

	clockRealtime = 0
)

var monotonicEpoch = time.Now()

// instantiateWASI builds the wasi_snapshot_preview1 module of one instance: the calls the Go runtime makes are
// implemented on the virtual descriptors of the instance, and every other import of the guest fails with ENOSYS.
// There is no file system: the guest only sees its standard output and error, and the descriptors of the host.
func (inst *instance) instantiateWASI(ctx context.Context, rt wazero.Runtime, compiled wazero.CompiledModule) error {
	envs := inst.environ()

	impl := map[string]api.GoModuleFunc{
		"args_sizes_get": func(_ context.Context, m api.Module, s []uint64) {
			writeU32(m, s[0], 0)
			writeU32(m, s[1], 0)
			s[0] = uint64(errnoSuccess)
		},
		"args_get": func(_ context.Context, _ api.Module, s []uint64) {
			s[0] = uint64(errnoSuccess)
		},
		"environ_sizes_get": func(_ context.Context, m api.Module, s []uint64) {
			size := 0
			for _, env := range envs {
				size += len(env) + 1
			}
			writeU32(m, s[0], uint32(len(envs)))
			writeU32(m, s[1], uint32(size))
			s[0] = uint64(errnoSuccess)
		},
		"environ_get": func(_ context.Context, m api.Module, s []uint64) {
			ptrs, buf := uint32(s[0]), uint32(s[1])
			for i, env := range envs {
				m.Memory().WriteUint32Le(ptrs+uint32(i)*4, buf)
				m.Memory().WriteString(buf, env)
				m.Memory().WriteByte(buf+uint32(len(env)), 0)
				buf += uint32(len(env)) + 1
			}
			s[0] = uint64(errnoSuccess)
		},
		"clock_time_get": func(_ context.Context, m api.Module, s []uint64) {
			ns := int64(time.Since(monotonicEpoch))
			if uint32(s[0]) == clockRealtime {
				ns = time.Now().UnixNano()
			}
			m.Memory().WriteUint64Le(uint32(s[2]), uint64(ns))
			s[0] = uint64(errnoSuccess)
		},
		"random_get": func(_ context.Context, m api.Module, s []uint64) {
			buf, ok := m.Memory().Read(uint32(s[0]), uint32(s[1]))
			if !ok {
				s[0] = uint64(errnoInval)

				return
			}
			_, _ = rand.Read(buf)
			s[0] = uint64(errnoSuccess)
		},
		"sched_yield": func(_ context.Context, _ api.Module, s []uint64) {
			s[0] = uint64(errnoSuccess)
		},
		"fd_prestat_get": func(_ context.Context, _ api.Module, s []uint64) {
			s[0] = uint64(errnoBadf)
		},
		"proc_exit": func(ctx context.Context, m api.Module, s []uint64) {
			code := uint32(s[0])
			_ = m.CloseWithExitCode(ctx, code)
			panic(sys.NewExitError(code))
		},
		"fd_fdstat_get": func(_ context.Context, m api.Module, s []uint64) {
			var stat [24]byte
			stat[0] = filetypeChar
			// A non-blocking descriptor makes the Go runtime wait for it with poll_oneoff instead of failing on EAGAIN.
			if uint32(s[0]) >= fdListener {
				stat[0] = filetypeStream
				binary.LittleEndian.PutUint16(stat[2:], fdflagNonblock)
			}
			binary.LittleEndian.PutUint64(stat[8:], ^uint64(0))
			binary.LittleEndian.PutUint64(stat[16:], ^uint64(0))
			m.Memory().Write(uint32(s[1]), stat[:])
			s[0] = uint64(errnoSuccess)
		},
		"fd_fdstat_set_flags": func(_ context.Context, _ api.Module, s []uint64) {
			s[0] = uint64(errnoSuccess)
		},
		"fd_close": func(_ context.Context, _ api.Module, s []uint64) {
			inst.closeFD(uint32(s[0]))
			s[0] = uint64(errnoSuccess)
		},
		"fd_write": func(_ context.Context, m api.Module, s []uint64) {
			s[0] = uint64(inst.fdWrite(m, uint32(s[0]), uint32(s[1]), uint32(s[2]), uint32(s[3])))
		},
		"fd_read": func(_ context.Context, m api.Module, s []uint64) {
			s[0] = uint64(inst.fdRead(m, uint32(s[0]), uint32(s[1]), uint32(s[2]), uint32(s[3])))
		},
		"poll_oneoff": func(ctx context.Context, m api.Module, s []uint64) {
			s[0] = uint64(inst.pollOneoff(ctx, m, uint32(s[0]), uint32(s[1]), uint32(s[2]), uint32(s[3])))
		},
	}

	builder := rt.NewHostModuleBuilder(wasiModuleName)
	for _, def := range compiled.ImportedFunctions() {
		module, name, _ := def.Import()
		if module != wasiModuleName {
			continue
		}

		fn := impl[name]
		if fn == nil {
			fn = func(_ context.Context, _ api.Module, s []uint64) {
				inst.logger.Debug().Str("function", name).Msg("Unsupported WASI call")
				if len(s) > 0 {
					s[0] = uint64(errnoNosys)
				}
			}
		}

		builder.NewFunctionBuilder().WithGoModuleFunction(fn, def.ParamTypes(), def.ResultTypes()).Export(name)
	}

	_, err := builder.Instantiate(ctx)

	return err
}

func (inst *instance) environ() []string {
	envs := make([]string, 0, len(inst.cfg.Envs))
	for _, name := range inst.cfg.Envs {
		envs = append(envs, name+"="+os.Getenv(name))
	}

	return envs
}

func (inst *instance) fdWrite(m api.Module, fd, iovs, iovsLen, resultPtr uint32) uint32 {
	var out *vfd
	if fd > 2 {
		if out = inst.fd(fd); out == nil {
			return errnoBadf
		}
	}

	total := uint32(0)
	for i := range iovsLen {
		ptr, _ := m.Memory().ReadUint32Le(iovs + i*8)
		size, _ := m.Memory().ReadUint32Le(iovs + i*8 + 4)
		data, ok := m.Memory().Read(ptr, size)
		if !ok {
			return errnoInval
		}

		if out == nil {
			inst.output(fd, data)
			total += size

			continue
		}

		n, errno := out.write(data)
		total += uint32(n)
		if errno != errnoSuccess {
			if total > 0 {
				break
			}

			return errno
		}
		if n < len(data) {
			break
		}
	}

	m.Memory().WriteUint32Le(resultPtr, total)

	return errnoSuccess
}

func (inst *instance) fdRead(m api.Module, fd, iovs, iovsLen, resultPtr uint32) uint32 {
	in := inst.fd(fd)
	if in == nil {
		return errnoBadf
	}

	total := uint32(0)
	for i := range iovsLen {
		ptr, _ := m.Memory().ReadUint32Le(iovs + i*8)
		size, _ := m.Memory().ReadUint32Le(iovs + i*8 + 4)
		buf, ok := m.Memory().Read(ptr, size)
		if !ok {
			return errnoInval
		}

		n, errno := in.read(buf)
		if errno != errnoSuccess {
			if total > 0 {
				break
			}

			return errno
		}
		total += uint32(n)
		if n < len(buf) {
			break
		}
	}

	m.Memory().WriteUint32Le(resultPtr, total)

	return errnoSuccess
}

// pollOneoff is the event loop of the instance: it blocks the instance goroutine until a subscribed descriptor is
// ready or the clock subscription expires. Every event the host has for the guest makes a descriptor ready and wakes
// the instance.
func (inst *instance) pollOneoff(ctx context.Context, m api.Module, in, out, subsLen, resultPtr uint32) uint32 {
	type subscription struct {
		userdata uint64
		kind     byte
		fd       uint32
	}

	subs := make([]subscription, 0, subsLen)
	var timer <-chan time.Time
	for i := range subsLen {
		base := in + i*48
		userdata, _ := m.Memory().ReadUint64Le(base)
		kind, _ := m.Memory().ReadByte(base + 8)

		sub := subscription{userdata: userdata, kind: kind}
		switch kind {
		case eventClock:
			timeout, _ := m.Memory().ReadUint64Le(base + 24)
			t := time.NewTimer(time.Duration(timeout))
			defer t.Stop()
			timer = t.C
		case eventFDRead, eventFDWrite:
			sub.fd, _ = m.Memory().ReadUint32Le(base + 16)
		default:
			return errnoInval
		}
		subs = append(subs, sub)
	}

	writeEvent := func(i int, sub subscription, errno uint32, nbytes uint64) {
		base := out + uint32(i)*32
		m.Memory().WriteUint64Le(base, sub.userdata)
		m.Memory().WriteUint16Le(base+8, uint16(errno))
		m.Memory().WriteByte(base+10, sub.kind)
		m.Memory().WriteUint64Le(base+16, nbytes)
		m.Memory().WriteUint16Le(base+24, 0)
	}

	for {
		n := 0
		for _, sub := range subs {
			if sub.kind == eventClock {
				continue
			}

			f := inst.fd(sub.fd)
			switch {
			case f == nil:
				writeEvent(n, sub, errnoBadf, 0)
				n++
			case sub.kind == eventFDRead:
				if size := f.readable(); size > 0 {
					writeEvent(n, sub, errnoSuccess, uint64(size))
					n++
				}
			case f.writable():
				writeEvent(n, sub, errnoSuccess, pipeSize)
				n++
			}
		}
		if n > 0 {
			m.Memory().WriteUint32Le(resultPtr, uint32(n))

			return errnoSuccess
		}
		if timer == nil && len(subs) == 0 {
			return errnoInval
		}

		select {
		case <-inst.wake:
		case <-timer:
			for _, sub := range subs {
				if sub.kind == eventClock {
					writeEvent(0, sub, errnoSuccess, 0)
					m.Memory().WriteUint32Le(resultPtr, 1)

					return errnoSuccess
				}
			}
		case <-inst.closed:
			// The instance is closing: stop the guest where it waits.
			_ = m.CloseWithExitCode(ctx, 0)
			panic(sys.NewExitError(0))
		}
	}
}

func writeU32(m api.Module, ptr uint64, v uint32) {
	m.Memory().WriteUint32Le(uint32(ptr), v)
}
