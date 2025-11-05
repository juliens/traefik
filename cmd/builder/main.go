package main

import _ "embed"

//go:embed go.mod
var goMod []byte

//go:embed go.sum
var goSum []byte

func main() {

	// fmt.Println("builder started")
	//
	// http.ListenAndServe(":80", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
	// 	fmt.Println("CALLED")
	//
	// 	plugin := req.URL.Query().Get("plugin")
	// 	if plugin == "" {
	// 		http.Error(rw, "plugin required", http.StatusBadRequest)
	// 		return
	// 	}
	// 	fmt.Println("/bin/bash", "-c", "./script/build-plugin.sh", plugin)
	// 	cmd := exec.Command("/bin/bash", "-c", "./script/build-plugin.sh "+plugin)
	// 	output, err := cmd.CombinedOutput()
	//
	// 	if err != nil {
	// 		fmt.Printf("plugin executable failed: %s\n%s", err, output)
	// 		http.Error(rw, "plugin execution failed", http.StatusInternalServerError)
	// 		return
	// 	}
	//
	// 	fmt.Printf("plugin executed: %s\n", string(output))
	//
	// 	file, err := os.Open("/dist/plugins.so")
	// 	if err != nil {
	// 		http.Error(rw, "Fichier non trouvé", http.StatusNotFound)
	// 		return
	// 	}
	// 	defer file.Close()
	//
	// 	// Obtenir les infos du fichier (optionnel, pour Content-Length)
	// 	fileInfo, err := file.Stat()
	// 	if err != nil {
	// 		http.Error(rw, "Erreur lecture fichier", http.StatusInternalServerError)
	// 		return
	// 	}
	//
	// 	// Headers
	// 	rw.Header().Set("Content-Type", "text/plain")
	// 	rw.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	//
	// 	// Copier le fichier vers le ResponseWriter
	// 	io.Copy(rw, file)
	// }))

}
