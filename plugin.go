package main

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"


    p77dd5ba6f05fc07e "github.com/tomMoulard/htransformation"

    pc711fcdd1738ce27 "github.com/0xanonymeow/traefik-request-filter"

    pcdcc39aec9984933 "github.com/0xanonymeow/traefik-token-auth"

    pca76ba6016c2d6dd "github.com/17media/plugin-allowpath"

    p6240bd9308738a4d "github.com/1cedsoda/traefik-umami-plugin"

    pf8c9cd332b5ad54e "github.com/23deg/jwt-middleware"

    pb43c211b54b62440 "github.com/3rd1t/traefik_login_authorization"

    p2cfa76d6710c93e4 "github.com/aarlint/pathauth"

    p3cf7b93f06ae6cd0 "github.com/acouvreur/sablier/plugins/traefik"

    p3c9b0c1f2a152f6a "github.com/acouvreur/traefik-modsecurity-plugin"

    p2a4700b0ec48bfd2 "github.com/acouvreur/traefik-ondemand-plugin"

    p561fae3b256a18c8 "github.com/AdamEszes/traefik-custom-headers-plugin"

    p6c20e4bbb07ff82f "github.com/adyanth/header-transform"

    pbac546f1505838be "github.com/agence-gaya/traefik-plugin-blockuseragent"

    pcde7270c8e96248 "github.com/agence-gaya/traefik-plugin-cloudflare"

    pcd6470b86b8d74a1 "github.com/agilezebra/jwt-middleware"

    p534ffc93fcd90782 "github.com/ajinkyak423/uiddemo"

    p3a29aeda02b8279c "github.com/albttx/traefik-plugin-sec-hasura"

    pe6bf4af386fe1cbb "github.com/alessandrolomanto/grpc-blocker"

    p5e4dce694092d709 "github.com/alessandrolomanto/plugin-simplecache"

    p3fd35178500dc4e1 "github.com/alex-held/traefik-plugin-rerouter"

    pf40afe4f5ba95b38 "github.com/alexandrebouthinon/traefik-kuzzle-auth"

    pa0e4072e40770cf "github.com/alexandreh2ag/traefik-ipfilter-basicauth"

    p95f154f5016b3a9b "github.com/alexandrovas/traefik-plugin-torblock"

    pe58740322f6ce251 "github.com/Amadeus331/cloudflarewarp"

    pa89ac4906b8da6f0 "github.com/amj1985/traefik-unleash-plugin"

    pa2d4f826d802d692 "github.com/ananace/traefik-fix-rgw"

    pa4507256a87348e3 "github.com/andrewkroh/google-oidc-auth-middleware"

    p1391895cf654d97d "github.com/antoniomacri/traefik-method-whitelist"

    pfeada453f96b2906 "github.com/apwe/headerproxy"

    p410bac2fac07528f "github.com/argyle-engineering/copy-header-value-traefik-plugin"

    pd339fb342fd7d8c9 "github.com/argyle-engineering/headerhasher"

    p68b381f3b60bb6c1 "github.com/argyle-engineering/traefik-ratelimiter-middleware"

    pd0b5716bcf63b06f "github.com/ArtemUgrimov/ResponseTimeBalancer"

    p665d044c2b3db7a7 "github.com/arwoosa/header2post"

    p678239b9b5ee65f1 "github.com/arwoosa/turnstile"

    p73266a06b2cabb18 "github.com/aseara/jc2h"

    p51a1fc821035f126 "github.com/astappiev/traefik-umami-feeder"

    p5bf4ae3b5b589d5b "github.com/atidev/traefikretryplugin"

    p79b465a67e071ca8 "github.com/aveq-research/requestfilter"

    pa6d8b2c1f3bbfa1 "github.com/axiaoxin/traefikplugindemo"

    paca572c644e2658d "github.com/axyi/traefik-query-append-url"

    pc3f48701b0e26fa7 "github.com/badgeinc/traefikgeoip2badge"

    p43678f5df9802645 "github.com/barmaths/w3c-tracecontext-creator"

    p63f1e342f9ac2f81 "github.com/Baseflow/traefik_rpthandler"

    p5a0e3203cd9eb31a "github.com/bay1ts/SiriusGeo"

    p595c8fa3df53084e "github.com/bcambl/keycloakopenid"

    p4bd3e625fddd536f "github.com/bchangiphc/normalizepath"

    pa2ac98f81746552a "github.com/Beanow/traefik-plugin-rawdata"

    pe0c7bec1b8a7d4d9 "github.com/behnambm/gors"

    ped95b33bc08c8d47 "github.com/benoitg31/traefik-forced-body-plugin"

    pe2fad426a46e82c5 "github.com/BetterCorp/cloudflarewarp"

    peed8e20c92e5d49a "github.com/beyerleinf/traefik-plugin-extract-cn"

    p6539c6cb1adceb60 "github.com/beyerleinf/traefik-plugin-rename-header"

    p850377991ae5c998 "github.com/Bigouden/headerguard"

    p42010707fa2441c4 "github.com/BilikoX/cloudflarewarp"

    pd63936cb3dbda410 "github.com/birotaio/traefik-plugins"

    p9915554052e2c794 "github.com/bitrvmpd/traefik-plugin-rewrite-headers"

    p32e95049bb14202a "github.com/bitzlato/traefik-telegram-ratelimiter"

    p21c909b4f11c9aeb "github.com/bjornharrtell/traefik-api-key-middleware3"

    p9f6deaf67e20fffb "github.com/bluecatengineering/traefik-aws-plugin"

    p5f31a9794a56af35 "github.com/blueshift-labs/traefik-block-regex-urls"

    p885c9df2f3f891b5 "github.com/bonovoxly/extractcookieregex"

    pc03323c73e4a7317 "github.com/bonsai-oss/custom-source-header"

    p87d0d089a7007 "github.com/bravepickle/traefik-change-response"

    p71c5a45c3fe94e93 "github.com/BrinkmannMi/traefik-auth-with-exceptions"

    pb666693622e83caf "github.com/brudnevskij/query2port"

    p77d566f96ec5fdaa "github.com/bukukasio/super-rate"

    pde2bf7b8e696a6dd "github.com/carnage-sh/sessionmapper"

    p2aa9cedf211bab32 "github.com/Catzilla/traefik-hydrate-headers"

    pf63079cd96e7d25e "github.com/Catzilla/traefik-jwt-internal"

    p7d6530cadcee9946 "github.com/cdwiegand/standard-security-headers-plugin"

    pcf188473459d48a9 "github.com/cdwiegand/traefik-add-trace-id-header-2"

    pb459eb3e19a1bd9e "github.com/cdwiegand/traefik-head-to-get"

    p9b71612dcb4b4537 "github.com/Ch1nkara/traefik-modsecurity-plugin"

    p9df400a5382731e4 "github.com/chahn/subfilter"

    p130a6260db7caa7c "github.com/chaitin/traefik-safeline"

    p89eb5310dbfd2350 "github.com/charanpreetp/fail2ban"

    p71940b1917feaf73 "github.com/che-incubator/header-rewrite-traefik-plugin"

    p5ed4c6bee9bd0d17 "github.com/chendo/traefik-guard"

    p77b33d323f7fe198 "github.com/chendo/traefik-request-shaper"

    pdfc9edd0bc78ce8 "github.com/chiztour/traefik-jwt-claims-header-plugin"

    p567c1c497b17f2a "github.com/chong19951021/token"

    pc59e0257a365164b "github.com/cilasbeltrame/lowestlatencyendpoint"

    p271b96aba84c6aeb "github.com/CitronusAcademy/traefik-maintenance-plugin"

    p419626a28d78c555 "github.com/clambin/traefik-throttler"

    p8992c3e480d000fc "github.com/Clasyc/tokenauth"

    p590b0935b87b7bea "github.com/ClimberJ/traefik-fail2ban-connector"

    p86aee1aabb564606 "github.com/clugg/traefik-enforce-header-case-plugin"

    pa9cd7c1ce7109e2a "github.com/cnmaple/yzjapidecryption"

    p3c211341f05b458e "github.com/conekta/header-based-proxy"

    p4b7e324e521c8833 "github.com/containeroo/duplicateheader"

    pe796367f4fc3f245 "github.com/cookielab/traefik-middleware-request-logger"

    p326a4ccb2c20b464 "github.com/corticph/queryparameter-to-header"

    p9e4ff126b442afca "github.com/craigbrogle/traefik-s3-plugin"

    pd67d4e3d85be6872 "github.com/crazygolem/traefik-subsonic-basicauth"

    pe6b629e19d6753b2 "github.com/credibil/pluginauth"

    p3492cc9d89d970f "github.com/csobrinho/traefik-plugin-s3-auth"

    p3a4054bd4c35afcc "github.com/ctrl-hub/traefik-auditor"

    pebf29ef77d346dd8 "github.com/Cubicroots-Playground/traefik-geoip-metrics-middleware"

    pc7dcc1dd7a095b44 "github.com/CumpsD/edns0"

    p13d8d8cb7a7912a6 "github.com/Cyb3r-Jak3/traefik-plugin-cloudflare"

    paad0be7297c02941 "github.com/danbiagini/traefik-cloud-saver"

    ped731d02e48f0d0 "github.com/danielbjornadal/traefik-cloudflare-plugin"

    p493613a416227a78 "github.com/daniels0056/traefik-simpleredirect"

    p8cbcffdc94924953 "github.com/dararish/captcha-protect"

    p79a57243a729bee8 "github.com/dariusandz/header-transmute"

    p30b60e7be766f883 "github.com/darkweak/go-esi/middleware/traefik"

    p54fd784626fb872f "github.com/dashpool/dashmiddleware"

    p3dd9c389d70878a2 "github.com/davewhit3/traefik-cf-device-detector"

    pcdeb999149103e5f "github.com/david-garcia-garcia/traefik-geoblock"

    p915df3171d91eeef "github.com/david-garcia-garcia/traefik-modsecurity"

    pc2d49d1a932e5422 "github.com/david-garcia-garcia/traefik-realip"

    pc3edcbcd4adcba86 "github.com/daxroc/traefik-jwt-org-redirect"

    pc2ba2ea112ba02b9 "github.com/dcasia/plugin-cond-redirect"

    p8cbccd42d646cb31 "github.com/dclairac/traefik-plugin-headers"

    pbc2ffb98e055a33f "github.com/decodeex/traefik_middleware"

    pc1ccefee96a8b186 "github.com/Desuuuu/traefik-cloudflare-plugin"

    pfec832e723e82f50 "github.com/Desuuuu/traefik-real-ip-plugin"

    pc20f6ff9f42f104b "github.com/dev-toolbox/traefik-plugin-parameters"

    p8d56739cef3fdf39 "github.com/developmentaid-org/denyip"

    p43fbefdb3b197eb1 "github.com/dgzlopes/traefik-datadog-event"

    pd012da2f6dbb4f3 "github.com/dgzlopes/traefik-fault-injection"

    pcbc8c3371b9d2983 "github.com/DIE-Bonn/MatomoTracking"

    pe8ad1623bb2bc858 "github.com/Dimoniq/jwtvalidator"

    p944c541f767590c3 "github.com/dimorder/dimdanredirect"

    p50325d2dc581fbc0 "github.com/DirtyCajunRice/subfilter"

    pfcb6a86df641353a "github.com/discoverygarden/traefik-ultimate-bad-bot-blocker"

    p7a623984ff72567b "github.com/dkijkuit/azurejwttokenvalidation"

    p5bd7650ad3f43f35 "github.com/dkijkuit/checkheadersplugin"

    pe9c5ccd2679f882d "github.com/dndll/header-to-queryparameter"

    p7048a7c357e8febc "github.com/dobots/multiplexer-proxy"

    p42746916597c3c76 "github.com/DogAndHerDude/plugin-aheadinator"

    pcd12a0ba08edf672 "github.com/domainesia/traefik-plugin-reformatheader"

    p70be20fc50f50704 "github.com/dominion-solutions/traefik-filter-on-field"

    pf36d0771cbb6b1bd "github.com/dragosnutu/traefik-plugin"

    pcbf0c0286cfd3a59 "github.com/dtomlinson91/traefik-api-key-middleware"

    pf9af7d864fd28cde "github.com/durvesh-palkar/traefik-epoch-header"

    pa804270def4ac4c4 "github.com/dzungmmp/host-header-plugin"

    p7fc433ce188edf54 "github.com/e-flux-platform/full-url-rewrite-traefik-plugin"

    pc218c96ea607897b "github.com/EasySolutionsIO/traefikxrequeststart"

    p2d01717153ed421b "github.com/ecov/traefik-plugin-introspect"

    p84e52dbfb24c5649 "github.com/edelbluth/tm_http_redirect"

    p753a9abaec686e97 "github.com/edelbluth/tm_no_ai_bots"

    p22430294162b32af "github.com/edgeflare/traefikopa"

    p1839341d301cce5b "github.com/edgeworx/static-response-plugin"

    pc8fcfabea557dac7 "github.com/elee1766/traefik-gubernator-plugin"

    p8e587c15275c044f "github.com/ELLIO-Technology/ELLIO-Traefik-Middleware-Plugin"

    pa9763b2bda45697d "github.com/emigrating/TraefikRealIPs"

    pd987e21b65277957 "github.com/esnunes/redirecterrors"

    pbc1f8238e736a205 "github.com/Evocelot/traefik-lazy-serve"

    pa15254f6a22bce9a "github.com/evolves-fr/traefik-plugin-redirect"

    peeebecb8f427029d "github.com/famedly/traefik-certauthz"

    pe3b8cafeddcc3d27 "github.com/Farrukhraz/headersvalidator"

    p30750a8907c53787 "github.com/Farrukhraz/jwt2headers"

    pcd75ee182eb8e017 "github.com/fdufault/mapheaders"

    p8b357f59d93a8b94 "github.com/feedzai/traefikRequestTimestamps"

    p57af4bb55a21399c "github.com/Felioh/traefik-post-to-delete"

    p2fabdf477ff1fb70 "github.com/filipmiik/traefik-standard-proxy-headers"

    pcf8525f4f548fe30 "github.com/FinalCAD/TraefikRegionalPlugin"

    pd50e7e1bc9a14b38 "github.com/firstred/crowdsec-bouncer-traefik-plugin"

    pc75a72fe73be7b7c "github.com/fliot/fail2ban"

    pcb4a2e6741ccbf78 "github.com/florentinl/websitepathconverter"

    p54286df88f888b55 "github.com/FlowingSPDG/traefik-plugin-pick-response-json"

    pfb85d5c37ae33523 "github.com/FlowingSPDG/traefik-plugin-query-to-json"

    p3be782b5ca5cc675 "github.com/fma965/cloudflarewarp"

    p64764d5564df6c2d "github.com/fopina/traefik-commonname-validator-plugin"

    p39fdb68b2c2c032f "github.com/formancehq/gateway-plugin-auth"

    p9add789cae89267d "github.com/fosrl/badger"

    pcee99c7126bf02dc "github.com/FoxxMD/traefik-rybbit-feeder"

    p94a67f7cfb4597f3 "github.com/frankforpresident/traefik-plugin-validate-headers"

    pacc4f657fc50a24 "github.com/fzoli/traefiklogger"

    pa232bc2f8777e8b0 "github.com/gaborini/traefik-header-rename-plugin"

    p33aec9e7e8ddd85a "github.com/galaxias/traefik-vault-auth"

    peaa48e445796cf49 "github.com/georg-jung/github-webhook-middleware"

    pcef173d3ad7a6d1f "github.com/germangorodnev/multi-jwt-validation-middleware"

    p77739c7a1b7ea45c "github.com/ghnexpress/traefik-cache"

    pea9acb1c37d051cc "github.com/ghnexpress/traefik-ratelimit"

    p6f0bae79b43254bd "github.com/giacomoferretti/add-missing-headers"

    p2f76229b2b19466e "github.com/GiGInnovationLabs/traefikgeoip2"

    pd551b66433983b50 "github.com/GiGInnovationLabs/traefikuseragent"

    p7dc8234e42a6f197 "github.com/glefer/sensitive-files-blocker"

    pd30b6ca553efb2ef "github.com/GLMONTER/crlchecker"

    p2b02da3ef7c83ba8 "github.com/golubev-ml/white-elephant"

    p13ec94e5ac60108a "github.com/gpabijan88/traefik-auth-middleware"

    p20f3e33ce2ef1871 "github.com/Gwojda/keycloakopenid"

    pdf4be6c472785b45 "github.com/h3xsh/traefik-middleware"

    peda8636352895044 "github.com/hajimAIM/badger"

    p2ce8d1268923496 "github.com/helpshift/timestamp-injector"

    p982e34b8d9ee5cd8 "github.com/hhftechnology/bandwidthlimiter"

    pfa7d0692f35dfdaf "github.com/hhftechnology/ipwhitelistshaper"

    p13965f73a02236f "github.com/hhftechnology/statiq"

    p58395cca998bb74c "github.com/hhftechnology/tailscale-access"

    p3e27f878d2ba5b60 "github.com/hhftechnology/traefik-queue-manager"

    p891f9ecdeb407f2 "github.com/hiasr/forwardmiddleware"

    p44688d2249d7989c "github.com/hiasr/jwtfieldsheader"

    p6c0539f8e449a394 "github.com/holysoles/bot-wrangler-traefik-plugin"

    pfa04a697ab1098e6 "github.com/Hongbo-Miao/traefik-plugin-disable-graphql-introspection"

    p9a6f63b2bc604777 "github.com/honghainguyen777/traefik-modsecurity-plugin"

    p62e73c0a9f22b92 "github.com/hukumonline-com/traefik-modifier-plugin"

    p9162e5b5a57cbadf "github.com/Hvid/proxyHeaders"

    paaf2f67559976b34 "github.com/hwmland/dbgdump"

    pe9094b35cb141947 "github.com/iamolegga/plugin-simplecache"

    pce823b8b679da645 "github.com/igoooor/conteo-traefik-cache"

    p219f1b52747c8773 "github.com/igoooor/plugin-simplecache-conteo"

    p571c122121196101 "github.com/ijo42/corsmiddleware"

    paec38f5a7bb893c8 "github.com/iloahz/traefik-plugin-manual-access-control/plugin"

    pb0c92171b6df5629 "github.com/im-kulikov/traefik-provider"

    peea659e6e4be8b63 "github.com/Imaskiller/ddns-allowlist"

    pd0db4d20eef66f05 "github.com/imKota/traefik-maintenance-warden"

    p3e8f0cbedb8b0f7e "github.com/imKota/traefik-maintenance"

    p9735a08c92f9657c "github.com/inalbilal/traefik-cookie-auth"

    pdc7e0b4976f5bd23 "github.com/indivisible/redirecterrors"

    pa1e30d8a617f6d43 "github.com/init-object/redirect-ipv6"

    p23a012e62d4329f5 "github.com/inventi/traefik-header-transmute"

    p18cd20703325be7a "github.com/invit/traefik-cloudfront"

    p4da3e3d0e2d9f8cc "github.com/iolabs-ag/traefik-throttle"

    p35d02e9f39896fa1 "github.com/ion-toolbox/traefik-jwt-eddsa"

    p9149cd0d96c8898c "github.com/irotem/jwt-rewrite"

    p266df9345eb8c41c "github.com/isotes/traefik-outgoing-oauth2-cc"

    pd681a5a74f269fcc "github.com/itninja04/traefik-gelf-plugin"

    p76884fad388d888 "github.com/iYUYUE/traefik-forwarded-real-ip"

    pf8660777e2f7b671 "github.com/J4NS-R/traefik-oauth-upstream"

    p47d0b143dc82232a "github.com/jacekstarondiscovery/traefik-redirector"

    p3c43d388dec7b577 "github.com/jackhillman/pluginesi"

    pf33fdfbabe9207ec "github.com/Jakob3xD/checkheaders"

    p73cfb2628ab7a001 "github.com/jamesmcroft/traefik-plugin-return-response"

    p8a090d24f5aa6024 "github.com/jamesmcroft/traefik-plugin-rewrite-response-headers"

    p26fee8d065b0c774 "github.com/jaybubs/headerdump"

    pf0249d0ea5b742d6 "github.com/jdel/staticresponse"

    p3546926426952a37 "github.com/jeessy2/traefik-ip2region"

    pf7964fc3d99c5064 "github.com/jeppestaerk/traefik-xff-to-xrealip"

    p2894b10cf8a3ebcb "github.com/jerrywoo96/AddForwardedHeader"

    pb539dcbbcc20012 "github.com/jerrywoo96/AddMissingHeaders"

    pf16b169502d7709e "github.com/jetersen/traefik-cloudfront-xforwarded"

    pcd4907694dda8483 "github.com/jghaanstra/badger"

    p2ccaa8c271c0ad4a "github.com/jiangwennn/traefik-plugin-ip2location-redirect"

    pa45ad1924fb8b0d0 "github.com/JimCronqvist/traefik-api-key-auth"

    pc0d717edfc425321 "github.com/JimmyTsai16/cloudflarewarp"

    pba597edd138da595 "github.com/JimmyTsai16/MethodAllowed"

    p1a7cfbb11afe3d58 "github.com/jmcarbo/keycloakopenid"

    p624c50f1ce964cab "github.com/joegarb/traefik-throttle"

    p9c0e0c7786e5877d "github.com/joinrepublic/traefik-csp-middleware"

    p1780b1f32b3cc296 "github.com/JonasSchubert/traefik-allow-countries"

    p29813527b2f32857 "github.com/JonasSchubert/traefik-block-paths"

    pfc77208eb17b58e2 "github.com/jonathaanhs/traefik-forward-auth-body"

    p15b21f8faa6921da "github.com/JoshuaBowerman/TraefikCookiePathReplacement"

    p330cd97b7796f89e "github.com/joy2fun/traefik-plugin-log-request"

    pca9e90a96a2f87e7 "github.com/jpxd/torblock"

    p961124afb2837b51 "github.com/jramsgz/traefik-real-ip"

    p7e6a1f5e51ac9eb0 "github.com/jrg1381/smrequestid"

    p72ef0dd1e0dc89df "github.com/Ju0x/traefik-security-txt"

    pcec35f4f564a11bb "github.com/juitde/traefik-plugin-fail2ban"

    p76c5b6318bc9bb31 "github.com/K8Trust/authcookie"

    p45cf6dafefaa00c8 "github.com/K8Trust/traefikwsbalancer"

    pcd4264c7d7361a5c "github.com/kahf-infra/traefikPluginPathHeader"

    pcadd02ceba82a753 "github.com/kaitencloud/traefik-svix-plugin"

    p391899cdc83d6c23 "github.com/KaloyanYosifov/traefik-plugin-insert-custom-header"

    p59069a2867e3a69e "github.com/kav789/plugindemo"

    pcf642a462056a768 "github.com/Kazyini/yatmp"

    p3e99f1e622944066 "github.com/kevtainer/denyip"

    pf62417f6800f1d24 "github.com/killer-djon/traefik-correlation"

    pa9ef56fa8cdf2289 "github.com/killer-djon/traefik-http2amqp"

    pacc32800cbce2997 "github.com/kingjan1999/traefik-plugin-custom-mtls"

    pe08462bddb97fe47 "github.com/kingjan1999/traefik-plugin-exception-authbasic"

    p47cbc01a8d63abc6 "github.com/kingjan1999/traefik-plugin-query-modification"

    pa4b4a6befb6268a8 "github.com/KizzyCode/mtlsrules-traefik-golang"

    p9c16a7b41332f31a "github.com/Knight-7/addheader"

    pf3d331d2fe105e89 "github.com/KodeyThomas/traefik-oidc"

    p988a639bb29f2bc6 "github.com/Koellewe/traefik-oauth-upstream"

    p470214870f6c7858 "github.com/korteke/traefik-waiting-room"

    p907527345c6867e7 "github.com/kotalco/crossover-activity"

    pc1a02744e609d62 "github.com/kotalco/crossover-blacklist"

    pee23a928ee8f3089 "github.com/kotalco/crossover-cache"

    pa8841e60f65adcee "github.com/kotalco/crossover-limiter"

    p64526958e05912b0 "github.com/kotalco/crossover-managed"

    p366480a0d009070f "github.com/kotalco/crossover"

    p6d6f857588df89cf "github.com/krokodaws/traefik-hidden-auth"

    p1cd8a0da46eb74ce "github.com/kucjac/traefik-block-ua"

    p858af0290f4b17b5 "github.com/kucjac/traefik-plugin-geoblock"

    pf791b567886bdd99 "github.com/kumina/headers-by-request"

    pd2cb2aaa3f9ebfc7 "github.com/kumina/traefik-routing-plugin"

    pdf580154b6563075 "github.com/kuzzleio/traefik-header-transform"

    p929f2d61313fb955 "github.com/kvncrw/denyip"

    pe41aea04f9d0311e "github.com/kyaxcorp/traefikdisolver"

    pd3ce1205f6d69179 "github.com/kzmake/traefik-plugin-forward-request"

    p434f677f11c2cd76 "github.com/l4rm4nd/traefik-warp"

    p792336c8a3eef78b "github.com/Lambda-IT/traefik-plugin-cookie-flags"

    pb53439c689c23e9e "github.com/LASER-Yi/traefik-drop-connection"

    p4ab9f1c41e2fc96d "github.com/LaurenceJJones/password-protect-traefik-plugin"

    pe7791fd7814826ce "github.com/lazzio/cleanclientip"

    p6fe7848921f297f2 "github.com/leesalminen/kratos-session-extend"

    p79ee65b399e9297f "github.com/legege/jwt-validation-middleware"

    p9f2af4bafd61e63c "github.com/leonjza/trauth"

    p13253ec47513795 "github.com/Lepkem/traefik-plugin-response-code-override"

    pf8fa4936eba58097 "github.com/LeslieRan/pluginDemo"

    p908773e7f75204f4 "github.com/libops/captcha-protect"

    p716470db4ab7c39a "github.com/lifter-ai/auth-token-exchange-plugin"

    p26f636bd0a87ece5 "github.com/Lightirius/traefik-auth-converter"

    pcbe145b97ebcf256 "github.com/lion7/traefik-jwt-headers-plugin"

    pb7f95f2745facfac "github.com/LiquidLogicLabs/traefik-plugin-cors-regex"

    pdc7058c43c0e7e19 "github.com/livekit/traefik-readiness-plugin"

    pf7ad488f882ea159 "github.com/LiveOakLabs/traefik_middleware_sigv4"

    pea45fa87b1248d55 "github.com/lixianliang/traefik-plugin-return-response"

    pb941033aad5d2a62 "github.com/lleadbet/traefik-plugin-cache-by-route"

    pc9eb247d85877a4 "github.com/longbridgeapp/traefik-session-max-age"

    pf4741024bda5ad60 "github.com/louiscavalcante/easy-traefik-rate-limit-jwt"

    pc040f583cfec259 "github.com/lsz66/traefik-cas-plugin"

    p7f8c5512c9136a5d "github.com/luizfonseca/traefik-github-oauth-plugin"

    pd230459b0f4919c6 "github.com/lukas-r/traefik-subdomain-path-rewrite-plugin"

    pb081345803255220 "github.com/lukaszraczylo/traefikoidc"

    pa0274d7d826c58ce "github.com/LumensGroup/scanblock"

    pce921b3d2e48dd97 "github.com/luthfikw/traefik-real-ip-go"

    p796763078307d7b "github.com/LyuHe-uestc/traefik-plugin-check"

    p3627fd471bb8bce8 "github.com/LyuHe-uestc/traefik-plugin-ipblacklist"

    pcff558478f73050e "github.com/LyuHe-uestc/traefik-plugin-token-auth"

    pe4d496d156f52878 "github.com/m-riedel/traefik-plugin-redirect-on-status"

    pc305703915052bda "github.com/m4rc3l-h3/headermodifications"

    p6685fcd74b181edf "github.com/MadddinTribleD/traefikaggregator"

    p1e0ca064094e43a "github.com/madebymode/traefik-modsecurity-plugin"

    p62c90b618d41ceb "github.com/maiaaraujo5/traefik-jwt-claims"

    pf9f5d12596e4e3ff "github.com/MarkusJx/traefik-wol"

    p76628e3e1cef8950 "github.com/Maronato/traefik_geoip"

    p64ca573d55be393e "github.com/maxlerebourg/crowdsec-bouncer-traefik-plugin"

    p5cae4ccfe2fe5c43 "github.com/mayankkumar2/traefik-plugin-securum-exire"

    p8fffaddba7e6e481 "github.com/mdklapwijk/traefik-plugin-request-id"

    p67a35149ba3c0acc "github.com/mdouchement/geoblock"

    pad6d656de6677dca "github.com/Medad-ai/medad-jwt-middleware"

    p1d5d1c74b4b2775b "github.com/Medzoner/traefik-plugin-cors-preflight"

    pab026d61ddf42814 "github.com/melchor629/traefik-error-page"

    p31df7aecf93b1802 "github.com/Menfre01/conditionheader"

    p5a52434be1fb502 "github.com/MGDIS/traefik-apimanager-plugin"

    pef095dee062153ab "github.com/Miladbr/tlscrlchecker"

    pb9398ef1e2bf64e3 "github.com/milosdjurdjevic/traefik-deep-linking-middleware"

    p2421326a87ae9f6b "github.com/Miromani4/traefik-plugin-AdminAPI_WebUI"

    pd89a44e9beb72b7b "github.com/mlambda-net/secret-header"

    pb07e8d796f01b403 "github.com/mmpx12/traefik-secpath"

    p5d091ad606195c6b "github.com/mohamed-abdelrhman/traefikloggerbridge"

    pf27ff1bc1dbc0e2 "github.com/momayyez/authztraefikgateway"

    pe20a41c12f2a5bc "github.com/momayyez/traefikauthz"

    p4a3d7424fa61bb0f "github.com/moonlight8978/traefik-cloudflare-geoblock"

    p3f06b657559bee4c "github.com/moonlightwatch/MethodBlock"

    p22180835c1c47639 "github.com/moonlightwatch/referer"

    pddbb97c9e007955 "github.com/moonlightwatch/ReturnClientIP"

    p984593cfac712905 "github.com/Morozzzko/traefik-csp-middleware"

    p48a540bf8b8025bb "github.com/mrambossek/traefik-extraheaders"

    pcb9612c6ed0c518b "github.com/mrdrelar/traefik-plugin-rewriteheader"

    p910cc03d4a1c785a "github.com/mridang/traefik-superheader"

    p209253b91557c632 "github.com/MrNinso/statusdonrouters"

    pb4e306b64835f3a1 "github.com/msgbyte/traefik-tianji-plugin"

    pe33efeca394726e0 "github.com/mubashiroliyantakath/toi"

    pfa8259cfd86f64a8 "github.com/muhgumus/traefik-token-middleware"

    p939b231a906a460d "github.com/music-tribe/azadjwtvalidation"

    p2b2a64c7ab8851a1 "github.com/MuXiu1997/traefik-github-oauth-plugin"

    p69c6463c7b72a85 "github.com/n0m4dz/jwt-cors"

    pe212d46aa8e78f2a "github.com/n2jsoft-public-org/traefik-maintenance-plugin"

    p3308929b14214b21 "github.com/ndelta0/interactionverifier"

    p4a0e6a5eb81c240b "github.com/negasus/traefik-plugin-bridge"

    p53e4e22c469731 "github.com/negasus/traefik-plugin-ip2location"

    pb03802630afad5b1 "github.com/neggles/middleflare"

    pf35f61fe67c2b357 "github.com/NenoxAG/traefikrealip"

    pd8aa60efc058c16f "github.com/nermolaev/traefik-request-id-short"

    p40d44f72db759626 "github.com/nese/forwardcookie"

    p68812a0b5fe6c03b "github.com/NETCOREXT/traefik-plugin-response-cache-control"

    pb4d93615c0bfc3ac "github.com/Netsocs-Team/keycloakopenid"

    pafe414e47a8ec48f "github.com/Netsocs-Team/netsocsplugin"

    pb122cbea7a1b3e5e "github.com/Netsocs-Team/traefik-owasp-security"

    pba20f0c4e69d "github.com/Netvigie/traefik-json-body-validator"

    p95b73f28e84d19a8 "github.com/neuraflow-github/my-traefik-jwt-plugin"

    p84e256f618109450 "github.com/ngocdv86/plugin-rewritebody"

    pf74a39beb0e04de5 "github.com/ngocdv86/rate-limit"

    p4f0537174face82f "github.com/nhomchatgpt/headerblock"

    p6e7610081c50357f "github.com/NiklasPor/traefik-plugin-replace-query-regex"

    p99adb2b0247f21cd "github.com/nilskohrs/environmentheader"

    pf3908b3757db92af "github.com/nilskohrs/headerblock"

    pd65bb09b2e7e629d "github.com/nilskohrs/pathauth"

    p706e78ba34ab4fd6 "github.com/nilskohrs/regex2redirect"

    pd13b8fbb49d46f57 "github.com/nilskohrs/reproxied"

    p3a31d7994625b7a5 "github.com/nilskohrs/stripcookie"

    pae6ea5e7e867358a "github.com/Noahnut/replacePathRegex"

    pcdae6056c5880760 "github.com/noaHson86/signature-plugin"

    peff5071eeef35ab5 "github.com/NovinSystemCom/identityplugin"

    p1928035367dedded "github.com/nscuro/traefik-plugin-geoblock"

    paf1271f54f4fbf82 "github.com/NX211/traefik-proxmox-provider"

    p21f01d20e90a15dd "github.com/NX211/traefik-webfinger"

    p5a3a3b40224f98d2 "github.com/nzin/traefik-cluster-ratelimit"

    p5825f722c9865ad "github.com/omar-shrbajy-arive/headerauthentication"

    pe8b769d9ad8ec2fc "github.com/opaas-cloud/traefik-plugin-proxy-cookie"

    p9150e36ea73bcaf4 "github.com/openware/barongz"

    p96a63e9a9286a05a "github.com/packruler/rewrite-body"

    p33d04b4cd62e4007 "github.com/packruler/traefik-themepark"

    pfa6ad40c666df602 "github.com/paladium/traefikkeycloak"

    paa1cf5af57ddab7 "github.com/pamdigitek-doomo/traefik-jwt"

    pdd854c089543d1f5 "github.com/PandaWorker/traefik-upstream-when"

    p6d20c1589d2f550f "github.com/Papercast-Limited/epoch"

    p942d236640fca508 "github.com/PascalMinder/geoblock"

    pabc7bba81e2d3e9f "github.com/PatrickMi/body-forward-auth"

    p3e36be5d0844066b "github.com/paul-vautier/enieca"

    p37d4a04837503f7b "github.com/PaulLeRoux142/TorBlockRedirect"

    p2b4720f85cb5b4ec "github.com/pavankumar0143/traefik-lambdaauthorizer"

    p2c87f3bb7183eaf3 "github.com/pavankumar0143/traefik-lambdarequesttransformer"

    p1f774d5dba6db4de "github.com/pavankumar0143/traefik-lambdaresponsetransformer"

    p58f7121716c1b4dd "github.com/PavloZastavnyi/headerstransformation"

    p7ad222b5e1b4b78b "github.com/Paxxs/traefik-get-real-ip"

    pc59e4212810de2d3 "github.com/pdazcom/botdetector"

    p4fb28aefa785a324 "github.com/Penitence1992/traefik-ldap-plugin"

    pb88cf4f810b06767 "github.com/pierre-verhaeghe/traefik-replace-response-code"

    pf84d65a6e29af10b "github.com/PingThingsIO/traefik-jwt-group-access"

    pb67997b599b14d41 "github.com/pipe01/plugin-requestid"

    p18c09b501bfc27d5 "github.com/Pival81/keycloakopenid"

    pbe330377648f5a65 "github.com/pnxs/traefik-plugin-mtls-header"

    p2a4d577c55d0442e "github.com/poloyacero/headauth"

    p9981954b6bd1872f "github.com/PongDev/traefikbodytransform"

    pcb9779f11f59a70b "github.com/portbrella/traefik_whitelist"

    pb617fb8fcf2611e0 "github.com/portofrotterdam/environmentheader"

    p5bcf4c2db72b424c "github.com/portofrotterdam/environmentpathappender"

    p564ff97e39e1ce8a "github.com/portofrotterdam/headerblock"

    p5b63e3d053a0302 "github.com/portofrotterdam/pathauth"

    p8ea3a6b4720064a5 "github.com/portofrotterdam/regex2redirect"

    pee0a0fef92696f77 "github.com/portofrotterdam/reproxied"

    p5b116bd5f754ca84 "github.com/portofrotterdam/stripcookie"

    p3a31907a2dc7fb23 "github.com/portswigger-cloud/cloudfrontgate"

    pc1bf5d27f93762c9 "github.com/portswigger-cloud/requestsenderplugin"

    p84e82d5c03696d04 "github.com/PRIHLOP/traefik-body-rewrite"

    pefd75e9bd580a75a "github.com/programic/traefik-maintenance-plugin"

    p9fa1167301e95ba6 "github.com/project-echo/traefik-ocsp"

    p1280d9eefca9c249 "github.com/przemek-carma/w3c-traceparent-generator"

    p73dc56df3ffe6733 "github.com/PseudoResonance/cloudflarewarp"

    pa9b721c8b3c93527 "github.com/PseudoResonance/traefikerrorreplace"

    p7d845c03e236993 "github.com/psncius/traefik-api-middleware"

    pc88f3cb6abdf1d8b "github.com/pvalletbo/traefik-blocklist"

    pbd9895c860d1dde0 "github.com/pvalletbo/traefik-forwarded-real-ip"

    p2fdf76ce2145c822 "github.com/pvliesdonk/mtlsforward"

    pbe1731e22087a7a0 "github.com/pxxonline/traefik-plugin-cors"

    pa0c8fc75db8b2c08 "github.com/pyksid/cloudflarewarp"

    p5d98499a72392805 "github.com/pyrho/badgerheaders"

    pb788e338709619ef "github.com/quintinheard/traefik-cors/traefik"

    pe36d636ac3559a7e "github.com/quortex/traefik-responsebodyrewrite"

    pd999b002eb65af1 "github.com/quortex/traefik-responseheadersfilter"

    p8fcb624c8690d057 "github.com/qwercik/traefik-original-uri"

    pcf17fd66255f650a "github.com/qxsugar/request-dispatch"

    p819d703205fd6192 "github.com/qxsugar/request-mark"

    pf98836f5f6059090 "github.com/qxsugar/traefik-jwt-parser"

    pa5483386e15bf56a "github.com/r3nic1e/traefik-plugin-add-response-header"

    p42c3b65c6b878eb2 "github.com/rafal-slowik/traceparent-plugin"

    p503af66018e4063e "github.com/Rajabalian/ipclient"

    pb341ca3bd0670116 "github.com/Rau-N/DomainSentinel"

    pae6a54e4192c57ff "github.com/renanqts/xdpfail2ban"

    pc9caaca396dda32 "github.com/rhabichl/applicationgatewaywhitelist"

    pcf490a990232afff "github.com/Ridecell/traefik-token-checker"

    p5e339e5c331a7c2b "github.com/rinokadijk/traefik-api-key"

    p3bb88aaab1ae0b55 "github.com/rinokadijk/traefik-openai-header"

    p4e0c1092a500dada "github.com/RiskIdent/traefik-remoteaddr-plugin"

    p5d88a78ef087cca7 "github.com/RiskIdent/traefik-tls-headers-plugin"

    p8e0fd2112689537b "github.com/rjop-hccgt/traefik-forward-slash-redirector"

    pe5d3bb9d60f17b58 "github.com/rjop-hccgt/traefikpluginhcindex"

    p7fc777d3465e3f9 "github.com/rocdove/replacepathfromurlregex"

    p65f68000846c473b "github.com/romracer/traefik-get-real-ip"

    pb517e0631b589fba "github.com/RouxAntoine/reproxied"

    p958c256be0fa43d3 "github.com/RSS3-Network/gatewayflowcontroller"

    p2cf9bbfd1fd2f241 "github.com/russ-p/traefik-plugin-static-sites"

    pf78f3c90684ef7fb "github.com/Russia9/body-size-limit"

    pa41dd68ff53e1b51 "github.com/sablierapp/sablier/plugins/traefik"

    p7457e5b728f06a3d "github.com/sadaghiani/traefik-auth-middleware"

    p845fb125f88f36d4 "github.com/safing/plausiblefeeder"

    pfb0acf2c38e4b068 "github.com/safing/scanblock"

    p1cd37128322c621c "github.com/safing/tlsauth"

    p8e27bda7b3ebdbc4 "github.com/sagarrakshe/b64-header-parser"

    p51da206482a8d06a "github.com/saltyorg/cloudflarewarp"

    p737fa363d94e1024 "github.com/saman-jafari/correlation-id-traefik"

    pfbe4cc0d107f41f3 "github.com/samerbahri98/sigv4middleware"

    p453b8b3875b99c8 "github.com/sanderPostma/traefik-validate-jwt"

    pbdd00c6440ab73a1 "github.com/sasd13/traefik-keycloak-authorizer"

    p438ae091c0f46beb "github.com/sasd13/traefik-proxy-forward"

    p1c7269fb4651c8fe "github.com/sasd13/traefik-proxy-header"

    pc6f76e54ef6731ad "github.com/schackoa/replacepathfromurlregex"

    p39807db6be088b87 "github.com/SchmitzDan/traefik-plugin-cookie-path-prefix"

    p77f48e1c385a89ba "github.com/SchmitzDan/traefik-plugin-proxy-cookie"

    pc14b69c1ab6de0a1 "github.com/SchmitzDan/traefik-plugin-redirect-location"

    pf347bc57ae317a3d "github.com/scrazy77/customerrorsrewrite"

    p95771a3e7bf7078c "github.com/scrazy77/dragonfly2imgproxy"

    pb6ae1ae2ce01044a "github.com/scrazy77/plugin-simplecache-nocache"

    p1d971293c9758a13 "github.com/Sensedia/traefik-plugin-decompress"

    p4e2178d04d6c0c2d "github.com/Septima/traefik-api-key-auth"

    pa6f30c66d795bf15 "github.com/SergioFloresG/corsmiddleware"

    paf635593a48897fe "github.com/set-de/jwt-middleware"

    p75afbeb31ee3f53 "github.com/sevensolutions/traefik-oidc-auth/src"

    pe9ace569cefce718 "github.com/sevensolutions/traefik-plugin-structure-demo/src"

    pd3eb13e8699c1e40 "github.com/shantanugadgil/traefik-block-regex-urls"

    p82a2221a57835821 "github.com/ShaunVyxw/my_plugin"

    pd51dbe5489cb4790 "github.com/Shoggomo/traefik_dynamic_public_whitelist"

    p95bea78df0849304 "github.com/SimpaiX-net/traefik-guard"

    p6b46b694e2448dc "github.com/skynet2/traefik-fallback-plugin"

    pdf6a64b0c011d7d1 "github.com/slimani-dev/dynamichost"

    p64b24807a23098d4 "github.com/smerschjohann/mtlswhitelist"

    pdf6a887bfe40cfa2 "github.com/snapt/traefik-nova-plugin"

    p1d999d72b46bdf7b "github.com/softwaremastermind/defaultcspheader"

    pebe2f99c694e50db "github.com/solution-libre/traefik-plugin-robots-txt"

    p1075e2fe49e6934d "github.com/soulbalz/correlationid"

    p4835087f73fb154b "github.com/soulbalz/traefik-check-body"

    p822ad9b7eb8fea8e "github.com/soulbalz/traefik-real-ip"

    p59f320025e1cfdc6 "github.com/sp-jcberleur/xrequesttrace"

    p534c77c64403e1e "github.com/Spakl-io/shorty"

    p63149cb160890c2c "github.com/sproutmaster/TraefikIPRules"

    p521a168be3d470fd "github.com/sstoner/cloudflaregate"

    pc9de36c5de123c83 "github.com/stabelo/traefik-tracking-cookie"

    p4b593cdbb54d6846 "github.com/steveiliop56/tinyrobotsblock"

    p134eb94002c89824 "github.com/strigo/traefik-auth-middleware"

    p9f6e10ec13ca7ab4 "github.com/subotaii/traefik-plugin-addprefix-from-host"

    p4c466ab25601f7b4 "github.com/sunalwaysknows/redirect2https"

    p1f0bd9f0f33221 "github.com/supergoudvis116/regex-redirect-joule"

    p98c5c6a64d8d5c6d "github.com/suteqa/plugin_record"

    p3d0a396678cc4ba2 "github.com/sw360cab/cncftaeplugin"

    p7d7cab2fea196357 "github.com/SwissDataScienceCenter/cookiefilter"

    p55267e921164c7c2 "github.com/sysradium/traefik-request-signature-verifier"

    pbbe71717d493b297 "github.com/taskmedia/ddns-allowlist"

    p9632062027b1f09b "github.com/taskmedia/ddns-whitelist"

    p2d51f97b32991267 "github.com/tdilber/anouncy-traefik-plugin"

    pe857d533a3c7416c "github.com/TDL-Bewatec/traefikbodytransform"

    pa1b3b93e672dfb70 "github.com/team-carepay/traefik-jwt-plugin"

    p3bbebc73b1696bac "github.com/team-carepay/traefik-opa-plugin"

    p7899ba08037c8df7 "github.com/TechAlchemistry/traefik-maintenance-warden"

    p8e654159c8c4d52c "github.com/tgrosinger/obsidian-publish-traefik-middleware"

    p2c351dbf32dfa7a5 "github.com/the-ccsn/traefik-plugin-rewritebody"

    pa7337476a94a3cd1 "github.com/theoguidoux/cookiesmanager"

    p21804cd5b3ce5e7a "github.com/thiagotognoli/traefikgeoip"

    p6ac8c5a73011f480 "github.com/Thijmen/traefik-query-parameters-middleware"

    p72063f9727be73cd "github.com/Thijmen/traefik-remove-query-parameters-by-regex"

    pc87c2241fa1ffbf2 "github.com/TicketGenieIO/plugin_forwardedauth"

    p8678aebe2bb033c2 "github.com/tilak999/traefikplugin"

    p15f9da7fb302e52c "github.com/tkreiner/traefik-regex-block"

    p277091e08872b7af "github.com/tmpim/tmpauth-traefik"

    pbe5cdb2421ed3add "github.com/tnt-sbab/jwt-verifier"

    peadef30a10c3ee95 "github.com/tnt-sbab/token-translator"

    p53e83b240d802b7d "github.com/toanz/jwt-token"

    pb6d67c091f2e1a2d "github.com/toanz/traefik-plugin-add-response-header"

    p600e93237b61d443 "github.com/togettoyou/traefik-timer-plugin"

    p8001fc243a0e7be "github.com/tommoulard/fail2ban"

    pc93e97832d3e6291 "github.com/tomMoulard/traefik-plugin-waeb"

    p71133c721994b704 "github.com/tonyfud/traefikjwttoken"

    p2d0e7c7c3c6b290f "github.com/tpaulus/jwt-middleware"

    pbc3d3536ae746bc6 "github.com/Traceableai/traceableai_traefik_plugin"

    pe259452c5816c5c8 "github.com/traefik-contrib/noop"

    pe6c37f1e7f02a54f "github.com/traefik-plugins/traefik-jwt-plugin"

    p7a60b7ee5412953a "github.com/traefik-plugins/traefikgeoip2"

    p1695a899b982abc5 "github.com/traefik-plugins/traefikuseragent"

    p68117edcfbcf4706 "github.com/traefik/plugin-blockpath"

    p619dce1d4f0b1c30 "github.com/traefik/plugin-log4shell"

    pecd51dbd1979b8c5 "github.com/traefik/plugin-rewritebody"

    p2ad1e6a65fe3d90c "github.com/traefik/plugin-simplecache"

    p10de222177ac8e3d "github.com/traefik/plugindemo"

    pf88cb09c7b357f87 "github.com/traefik/pluginproviderdemo"

    pdb96acd5de7db9c4 "github.com/Treblle/TreblleTraefikPluginGo"

    p3bfef9eb9e515126 "github.com/TreyWW/traefik-plugin-original-host-header"

    pc4656518e1fa29b1 "github.com/TRIMM/redirects-traefik-middleware"

    pbc2d85eda113c436 "github.com/TRIMM/traefik-maintenance"

    pad3f4dc55fb58938 "github.com/trinnylondon/lowercase"

    p481b01b6785e6733 "github.com/trinnylondon/traefik-add-trace-id"

    pe04bc247cd53be62 "github.com/trois-six/plugin-httplog"

    paf5cb2d07c9d73e2 "github.com/trois-six/plugin-securelink"

    pa163bd1824ec3b6d "github.com/trolleksii/traefik-plugin-mutate-headers"

    p1a9a222a5219607e "github.com/trondhindenes/traefikreplay"

    p8b695bef12267275 "github.com/tuxgal/traefik_inline_response"

    p69de2b3c97d8c975 "github.com/unbasical/traefik-json-body2header"

    pb5ed6cb55532db95 "github.com/unnoo/forward-port"

    p8bd1b4be9ce24321 "github.com/unsoon/traefik-open-policy-agent"

    p58fb11807ab8c4f4 "github.com/unsoon/traefik-require-auth-headers"

    p1a0cafb0f33a0aba "github.com/usalko/swagger-merge-docs"

    p1be319526fc5b221 "github.com/usalko/swagger-ring"

    p7f2b622d496639e4 "github.com/Uscreen-video/traefik-plugin-rewritehost"

    p20256bde45d79155 "github.com/usegiam/giam-traefik-plugin"

    p6d77d5d507f5344a "github.com/v-electrolux/extractcookie"

    p58e152f0f7bf56bc "github.com/v-electrolux/http2grpc"

    p5eb07795cc09a201 "github.com/v-electrolux/tlsclientcertforward"

    p6b67549cfe479764 "github.com/valebedeva/convertheader"

    p4026bc342965301 "github.com/valksor/traefik-conditional-headers"

    p11dbe61bccca063 "github.com/VanagaS/charset-converter"

    p790bbe560b3730b "github.com/VanagaS/preflight-custom-headers"

    p9f9432d22da23bac "github.com/Vandebron/traefik-cloudflare-plugin"

    pf8867bec0324b389 "github.com/Vandebron/traefik-keycloak"

    p5d2fbb210f7239e3 "github.com/vaspapadopoulos/traefik-cookie-handler-plugin"

    pdd53d567a4695233 "github.com/vercel-saleseng/traefik-oidc-auth-plugin"

    p38be0987ed87a123 "github.com/vidiemme/accesscontrol-ip-or-header"

    p55c3c0b1fb6420c8 "github.com/vidosits/header-pattern-proxy"

    p15cd59cddf3e4a29 "github.com/vincentinttsh/cloudflareip"

    pe576660eb45c4da1 "github.com/vincentinttsh/rewriteheaders"

    pb66a9da48dc1ff18 "github.com/virtualzone/rewriteheaders"

    pf418760a6fdf67b3 "github.com/vitaly-erofeev/avanpost_jwt_modification"

    pa5f6a5675f4d8a1 "github.com/vnghia/traefik-plugin-rewrite-cookie-path"

    pd068c74d517a2596 "github.com/vslinko/secret-auth"

    p806c0df7c2d4c279 "github.com/vtacquet/redbase-plugin"

    p2152c0f1d7208d4a "github.com/Wafris/wafris-traefik"

    p2c86845eef676226 "github.com/WagnerPMC/reverseguard"

    p5a5faa7efbd63314 "github.com/WalterP/traefik-mtls-check-plugin"

    pdfcc4f318391e065 "github.com/wbpaygate/traefik-headers"

    p1c307dd9120b21a1 "github.com/wbpaygate/traefik-ratelimit"

    pedb7dfcb234a7775 "github.com/wdonne/traefikoidc"

    p464926337b0895a5 "github.com/wiltonsr/ldapAuth"

    p51aeb3800fffa746 "github.com/WithourAI/path-auth-redirector"

    pa2008ad345e09978 "github.com/worldline-go/traefik-plugin-hello"

    p851882b77bc0c090 "github.com/wzator/headerblock"

    p69b1d9e8d5ce7e56 "github.com/x-ream/traefik-plugin-jwt-antpath"

    pf6a25e65d1514146 "github.com/xabinapal/traefik-authentik-forward-plugin"

    pe39f61d858c0d73b "github.com/xabinapal/traefik-customizable-auth-forward-plugin"

    p4322cf273aeb6827 "github.com/XciD/traefik-plugin-rewrite-headers"

    pbe14feca14520c32 "github.com/xethlyx/traefik-real-ip"

    p32f8cd28c37ba820 "github.com/xmd3/traefik-cf-ip"

    pce56639e32815d93 "github.com/Yeicor/traefikgothauth"

    pe3b0133900ee20d4 "github.com/Yeicor/traefikoidc"

    p3c2585221ef448d3 "github.com/yoeluk/traefik-authz-plugin"

    pdc262956775441bd "github.com/yurasavin/traefiktimestampheader"

    p5886b671fae37a5a "github.com/zackzackzackzack/traefik_datadog_tracing"

    p4969a4947a45f2c8 "github.com/zalbiraw/custommetrics"

    p1b823c529776d33b "github.com/zalbiraw/formdata"

    p79a3c3471f6df16d "github.com/zalbiraw/headertoquery"

    pe0aed7bf48d556a4 "github.com/zalbiraw/jwtvalidator"

    p978e455bacce01f0 "github.com/zalbiraw/ociaitoopenai"

    p342d93e806e33745 "github.com/zalbiraw/ociauth"

    p8a9943efa7058db0 "github.com/zalbiraw/pcprovider"

    p6a0e4bc5fbbcb72 "github.com/zalbiraw/requesttemplate"

    p30e75bbc7aebb56d "github.com/zalbiraw/tokencounter"

    p6226bc26ee729ddc "github.com/zalbiraw/traefikprovider"

    pae98f24de4ec16b6 "github.com/zekihan/cloudflarewarp"

    p8fa41ff278640d29 "github.com/zekihan/traefik-rate-limit"

    pa69bd68632a6d4a "github.com/zekihan/traefik-real-ip"

    p425bfe4163da80fe "github.com/ZeroGachis/traefik-auth-middleware"

    p73ea40d9c940ae52 "github.com/ZeroGachis/traefik-block-terminated-clients"

    pa6c334d53ef87999 "github.com/ZeroGachis/traefik-magic-jwt"

    pe84af6ccaf83c1b6 "github.com/ZeroGachis/traefik-oauth"

    p6d2f7cca31cb119c "github.com/ZeroGachis/traefik-request-id"

    p653a7a3120d40757 "github.com/zhaohongyang0701/add-trace-response-header"

    pdbbb9395da8784b4 "github.com/zhaohongyang0701/scanblock"

    p4b1d3bc0d1dd8d0a "github.com/zhaohongyang0701/trace"

    pcb5a5763c003f300 "github.com/zorgzerg/traefik-s3-proxy-plugin"

    pd38a079e07f408ad "github.com/ztelliot/traefik-echoserver"

    p911d7a9db819e335 "github.com/zyeming/rejectcontries"

)

type plugin struct {
	Create any
	New    any
	Version string
}

func init() {
  LoadPlugins()
}

func LoadPlugins() {

	pluginMap["github.com/tomMoulard/htransformation"] = plugin{
		Create: p77dd5ba6f05fc07e.CreateConfig,
		New:    p77dd5ba6f05fc07e.New,
		Version: "v0.3.3",
	}

	pluginMap["github.com/0xanonymeow/traefik-request-filter"] = plugin{
		Create: pc711fcdd1738ce27.CreateConfig,
		New:    pc711fcdd1738ce27.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/0xanonymeow/traefik-token-auth"] = plugin{
		Create: pcdcc39aec9984933.CreateConfig,
		New:    pcdcc39aec9984933.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/17media/plugin-allowpath"] = plugin{
		Create: pca76ba6016c2d6dd.CreateConfig,
		New:    pca76ba6016c2d6dd.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/1cedsoda/traefik-umami-plugin"] = plugin{
		Create: p6240bd9308738a4d.CreateConfig,
		New:    p6240bd9308738a4d.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/23deg/jwt-middleware"] = plugin{
		Create: pf8c9cd332b5ad54e.CreateConfig,
		New:    pf8c9cd332b5ad54e.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/3rd1t/traefik_login_authorization"] = plugin{
		Create: pb43c211b54b62440.CreateConfig,
		New:    pb43c211b54b62440.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/aarlint/pathauth"] = plugin{
		Create: p2cfa76d6710c93e4.CreateConfig,
		New:    p2cfa76d6710c93e4.New,
		Version: "v0.2.3",
	}

	pluginMap["github.com/acouvreur/sablier/plugins/traefik"] = plugin{
		Create: p3cf7b93f06ae6cd0.CreateConfig,
		New:    p3cf7b93f06ae6cd0.New,
		Version: "v1.8.0",
	}

	pluginMap["github.com/acouvreur/traefik-modsecurity-plugin"] = plugin{
		Create: p3c9b0c1f2a152f6a.CreateConfig,
		New:    p3c9b0c1f2a152f6a.New,
		Version: "v1.3.0",
	}

	pluginMap["github.com/acouvreur/traefik-ondemand-plugin"] = plugin{
		Create: p2a4700b0ec48bfd2.CreateConfig,
		New:    p2a4700b0ec48bfd2.New,
		Version: "v1.3.0",
	}

	pluginMap["github.com/AdamEszes/traefik-custom-headers-plugin"] = plugin{
		Create: p561fae3b256a18c8.CreateConfig,
		New:    p561fae3b256a18c8.New,
		Version: "v0.1.4",
	}

	pluginMap["github.com/adyanth/header-transform"] = plugin{
		Create: p6c20e4bbb07ff82f.CreateConfig,
		New:    p6c20e4bbb07ff82f.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/agence-gaya/traefik-plugin-blockuseragent"] = plugin{
		Create: pbac546f1505838be.CreateConfig,
		New:    pbac546f1505838be.New,
		Version: "v0.1.8",
	}

	pluginMap["github.com/agence-gaya/traefik-plugin-cloudflare"] = plugin{
		Create: pcde7270c8e96248.CreateConfig,
		New:    pcde7270c8e96248.New,
		Version: "v1.2.0",
	}

	pluginMap["github.com/agilezebra/jwt-middleware"] = plugin{
		Create: pcd6470b86b8d74a1.CreateConfig,
		New:    pcd6470b86b8d74a1.New,
		Version: "v1.3.4",
	}

	pluginMap["github.com/ajinkyak423/uiddemo"] = plugin{
		Create: p534ffc93fcd90782.CreateConfig,
		New:    p534ffc93fcd90782.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/albttx/traefik-plugin-sec-hasura"] = plugin{
		Create: p3a29aeda02b8279c.CreateConfig,
		New:    p3a29aeda02b8279c.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/alessandrolomanto/grpc-blocker"] = plugin{
		Create: pe6bf4af386fe1cbb.CreateConfig,
		New:    pe6bf4af386fe1cbb.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/alessandrolomanto/plugin-simplecache"] = plugin{
		Create: p5e4dce694092d709.CreateConfig,
		New:    p5e4dce694092d709.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/alex-held/traefik-plugin-rerouter"] = plugin{
		Create: p3fd35178500dc4e1.CreateConfig,
		New:    p3fd35178500dc4e1.New,
		Version: "v0.0.9",
	}

	pluginMap["github.com/alexandrebouthinon/traefik-kuzzle-auth"] = plugin{
		Create: pf40afe4f5ba95b38.CreateConfig,
		New:    pf40afe4f5ba95b38.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/alexandreh2ag/traefik-ipfilter-basicauth"] = plugin{
		Create: pa0e4072e40770cf.CreateConfig,
		New:    pa0e4072e40770cf.New,
		Version: "v1.0.4",
	}

	pluginMap["github.com/alexandrovas/traefik-plugin-torblock"] = plugin{
		Create: p95f154f5016b3a9b.CreateConfig,
		New:    p95f154f5016b3a9b.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/Amadeus331/cloudflarewarp"] = plugin{
		Create: pe58740322f6ce251.CreateConfig,
		New:    pe58740322f6ce251.New,
		Version: "v1.3.4",
	}

	pluginMap["github.com/amj1985/traefik-unleash-plugin"] = plugin{
		Create: pa89ac4906b8da6f0.CreateConfig,
		New:    pa89ac4906b8da6f0.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/ananace/traefik-fix-rgw"] = plugin{
		Create: pa2d4f826d802d692.CreateConfig,
		New:    pa2d4f826d802d692.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/andrewkroh/google-oidc-auth-middleware"] = plugin{
		Create: pa4507256a87348e3.CreateConfig,
		New:    pa4507256a87348e3.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/antoniomacri/traefik-method-whitelist"] = plugin{
		Create: p1391895cf654d97d.CreateConfig,
		New:    p1391895cf654d97d.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/apwe/headerproxy"] = plugin{
		Create: pfeada453f96b2906.CreateConfig,
		New:    pfeada453f96b2906.New,
		Version: "v0.5.0",
	}

	pluginMap["github.com/argyle-engineering/copy-header-value-traefik-plugin"] = plugin{
		Create: p410bac2fac07528f.CreateConfig,
		New:    p410bac2fac07528f.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/argyle-engineering/headerhasher"] = plugin{
		Create: pd339fb342fd7d8c9.CreateConfig,
		New:    pd339fb342fd7d8c9.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/argyle-engineering/traefik-ratelimiter-middleware"] = plugin{
		Create: p68b381f3b60bb6c1.CreateConfig,
		New:    p68b381f3b60bb6c1.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/ArtemUgrimov/ResponseTimeBalancer"] = plugin{
		Create: pd0b5716bcf63b06f.CreateConfig,
		New:    pd0b5716bcf63b06f.New,
		Version: "v2.0.2",
	}

	pluginMap["github.com/arwoosa/header2post"] = plugin{
		Create: p665d044c2b3db7a7.CreateConfig,
		New:    p665d044c2b3db7a7.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/arwoosa/turnstile"] = plugin{
		Create: p678239b9b5ee65f1.CreateConfig,
		New:    p678239b9b5ee65f1.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/aseara/jc2h"] = plugin{
		Create: p73266a06b2cabb18.CreateConfig,
		New:    p73266a06b2cabb18.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/astappiev/traefik-umami-feeder"] = plugin{
		Create: p51a1fc821035f126.CreateConfig,
		New:    p51a1fc821035f126.New,
		Version: "v1.3.0",
	}

	pluginMap["github.com/atidev/traefikretryplugin"] = plugin{
		Create: p5bf4ae3b5b589d5b.CreateConfig,
		New:    p5bf4ae3b5b589d5b.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/aveq-research/requestfilter"] = plugin{
		Create: p79b465a67e071ca8.CreateConfig,
		New:    p79b465a67e071ca8.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/axiaoxin/traefikplugindemo"] = plugin{
		Create: pa6d8b2c1f3bbfa1.CreateConfig,
		New:    pa6d8b2c1f3bbfa1.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/axyi/traefik-query-append-url"] = plugin{
		Create: paca572c644e2658d.CreateConfig,
		New:    paca572c644e2658d.New,
		Version: "v0.0.9",
	}

	pluginMap["github.com/badgeinc/traefikgeoip2badge"] = plugin{
		Create: pc3f48701b0e26fa7.CreateConfig,
		New:    pc3f48701b0e26fa7.New,
		Version: "v0.0.11",
	}

	pluginMap["github.com/barmaths/w3c-tracecontext-creator"] = plugin{
		Create: p43678f5df9802645.CreateConfig,
		New:    p43678f5df9802645.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/Baseflow/traefik_rpthandler"] = plugin{
		Create: p63f1e342f9ac2f81.CreateConfig,
		New:    p63f1e342f9ac2f81.New,
		Version: "v0.1.5",
	}

	pluginMap["github.com/bay1ts/SiriusGeo"] = plugin{
		Create: p5a0e3203cd9eb31a.CreateConfig,
		New:    p5a0e3203cd9eb31a.New,
		Version: "v2.5.0",
	}

	pluginMap["github.com/bcambl/keycloakopenid"] = plugin{
		Create: p595c8fa3df53084e.CreateConfig,
		New:    p595c8fa3df53084e.New,
		Version: "v0.1.47",
	}

	pluginMap["github.com/bchangiphc/normalizepath"] = plugin{
		Create: p4bd3e625fddd536f.CreateConfig,
		New:    p4bd3e625fddd536f.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/Beanow/traefik-plugin-rawdata"] = plugin{
		Create: pa2ac98f81746552a.CreateConfig,
		New:    pa2ac98f81746552a.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/behnambm/gors"] = plugin{
		Create: pe0c7bec1b8a7d4d9.CreateConfig,
		New:    pe0c7bec1b8a7d4d9.New,
		Version: "1.0.0",
	}

	pluginMap["github.com/benoitg31/traefik-forced-body-plugin"] = plugin{
		Create: ped95b33bc08c8d47.CreateConfig,
		New:    ped95b33bc08c8d47.New,
		Version: "v0.1.4",
	}

	pluginMap["github.com/BetterCorp/cloudflarewarp"] = plugin{
		Create: pe2fad426a46e82c5.CreateConfig,
		New:    pe2fad426a46e82c5.New,
		Version: "v1.3.3",
	}

	pluginMap["github.com/beyerleinf/traefik-plugin-extract-cn"] = plugin{
		Create: peed8e20c92e5d49a.CreateConfig,
		New:    peed8e20c92e5d49a.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/beyerleinf/traefik-plugin-rename-header"] = plugin{
		Create: p6539c6cb1adceb60.CreateConfig,
		New:    p6539c6cb1adceb60.New,
		Version: "v1.1.1",
	}

	pluginMap["github.com/Bigouden/headerguard"] = plugin{
		Create: p850377991ae5c998.CreateConfig,
		New:    p850377991ae5c998.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/BilikoX/cloudflarewarp"] = plugin{
		Create: p42010707fa2441c4.CreateConfig,
		New:    p42010707fa2441c4.New,
		Version: "v1.3.4",
	}

	pluginMap["github.com/birotaio/traefik-plugins"] = plugin{
		Create: pd63936cb3dbda410.CreateConfig,
		New:    pd63936cb3dbda410.New,
		Version: "v0.3.0",
	}

	pluginMap["github.com/bitrvmpd/traefik-plugin-rewrite-headers"] = plugin{
		Create: p9915554052e2c794.CreateConfig,
		New:    p9915554052e2c794.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/bitzlato/traefik-telegram-ratelimiter"] = plugin{
		Create: p32e95049bb14202a.CreateConfig,
		New:    p32e95049bb14202a.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/bjornharrtell/traefik-api-key-middleware3"] = plugin{
		Create: p21c909b4f11c9aeb.CreateConfig,
		New:    p21c909b4f11c9aeb.New,
		Version: "v0.6.0",
	}

	pluginMap["github.com/bluecatengineering/traefik-aws-plugin"] = plugin{
		Create: p9f6deaf67e20fffb.CreateConfig,
		New:    p9f6deaf67e20fffb.New,
		Version: "v1.2.0",
	}

	pluginMap["github.com/blueshift-labs/traefik-block-regex-urls"] = plugin{
		Create: p5f31a9794a56af35.CreateConfig,
		New:    p5f31a9794a56af35.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/bonovoxly/extractcookieregex"] = plugin{
		Create: p885c9df2f3f891b5.CreateConfig,
		New:    p885c9df2f3f891b5.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/bonsai-oss/custom-source-header"] = plugin{
		Create: pc03323c73e4a7317.CreateConfig,
		New:    pc03323c73e4a7317.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/bravepickle/traefik-change-response"] = plugin{
		Create: p87d0d089a7007.CreateConfig,
		New:    p87d0d089a7007.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/BrinkmannMi/traefik-auth-with-exceptions"] = plugin{
		Create: p71c5a45c3fe94e93.CreateConfig,
		New:    p71c5a45c3fe94e93.New,
		Version: "v0.9.2",
	}

	pluginMap["github.com/brudnevskij/query2port"] = plugin{
		Create: pb666693622e83caf.CreateConfig,
		New:    pb666693622e83caf.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/bukukasio/super-rate"] = plugin{
		Create: p77d566f96ec5fdaa.CreateConfig,
		New:    p77d566f96ec5fdaa.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/carnage-sh/sessionmapper"] = plugin{
		Create: pde2bf7b8e696a6dd.CreateConfig,
		New:    pde2bf7b8e696a6dd.New,
		Version: "v0.3.3",
	}

	pluginMap["github.com/Catzilla/traefik-hydrate-headers"] = plugin{
		Create: p2aa9cedf211bab32.CreateConfig,
		New:    p2aa9cedf211bab32.New,
		Version: "v0.4.0",
	}

	pluginMap["github.com/Catzilla/traefik-jwt-internal"] = plugin{
		Create: pf63079cd96e7d25e.CreateConfig,
		New:    pf63079cd96e7d25e.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/cdwiegand/standard-security-headers-plugin"] = plugin{
		Create: p7d6530cadcee9946.CreateConfig,
		New:    p7d6530cadcee9946.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/cdwiegand/traefik-add-trace-id-header-2"] = plugin{
		Create: pcf188473459d48a9.CreateConfig,
		New:    pcf188473459d48a9.New,
		Version: "v0.31.3",
	}

	pluginMap["github.com/cdwiegand/traefik-head-to-get"] = plugin{
		Create: pb459eb3e19a1bd9e.CreateConfig,
		New:    pb459eb3e19a1bd9e.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/Ch1nkara/traefik-modsecurity-plugin"] = plugin{
		Create: p9b71612dcb4b4537.CreateConfig,
		New:    p9b71612dcb4b4537.New,
		Version: "v1.3.2",
	}

	pluginMap["github.com/chahn/subfilter"] = plugin{
		Create: p9df400a5382731e4.CreateConfig,
		New:    p9df400a5382731e4.New,
		Version: "v0.1.4",
	}

	pluginMap["github.com/chaitin/traefik-safeline"] = plugin{
		Create: p130a6260db7caa7c.CreateConfig,
		New:    p130a6260db7caa7c.New,
		Version: "v1.4.0",
	}

	pluginMap["github.com/charanpreetp/fail2ban"] = plugin{
		Create: p89eb5310dbfd2350.CreateConfig,
		New:    p89eb5310dbfd2350.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/che-incubator/header-rewrite-traefik-plugin"] = plugin{
		Create: p71940b1917feaf73.CreateConfig,
		New:    p71940b1917feaf73.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/chendo/traefik-guard"] = plugin{
		Create: p5ed4c6bee9bd0d17.CreateConfig,
		New:    p5ed4c6bee9bd0d17.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/chendo/traefik-request-shaper"] = plugin{
		Create: p77b33d323f7fe198.CreateConfig,
		New:    p77b33d323f7fe198.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/chiztour/traefik-jwt-claims-header-plugin"] = plugin{
		Create: pdfc9edd0bc78ce8.CreateConfig,
		New:    pdfc9edd0bc78ce8.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/chong19951021/token"] = plugin{
		Create: p567c1c497b17f2a.CreateConfig,
		New:    p567c1c497b17f2a.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/cilasbeltrame/lowestlatencyendpoint"] = plugin{
		Create: pc59e0257a365164b.CreateConfig,
		New:    pc59e0257a365164b.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/CitronusAcademy/traefik-maintenance-plugin"] = plugin{
		Create: p271b96aba84c6aeb.CreateConfig,
		New:    p271b96aba84c6aeb.New,
		Version: "v0.1.11",
	}

	pluginMap["github.com/clambin/traefik-throttler"] = plugin{
		Create: p419626a28d78c555.CreateConfig,
		New:    p419626a28d78c555.New,
		Version: "v0.2.1",
	}

	pluginMap["github.com/Clasyc/tokenauth"] = plugin{
		Create: p8992c3e480d000fc.CreateConfig,
		New:    p8992c3e480d000fc.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/ClimberJ/traefik-fail2ban-connector"] = plugin{
		Create: p590b0935b87b7bea.CreateConfig,
		New:    p590b0935b87b7bea.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/clugg/traefik-enforce-header-case-plugin"] = plugin{
		Create: p86aee1aabb564606.CreateConfig,
		New:    p86aee1aabb564606.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/cnmaple/yzjapidecryption"] = plugin{
		Create: pa9cd7c1ce7109e2a.CreateConfig,
		New:    pa9cd7c1ce7109e2a.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/conekta/header-based-proxy"] = plugin{
		Create: p3c211341f05b458e.CreateConfig,
		New:    p3c211341f05b458e.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/containeroo/duplicateheader"] = plugin{
		Create: p4b7e324e521c8833.CreateConfig,
		New:    p4b7e324e521c8833.New,
		Version: "v1.0.26",
	}

	pluginMap["github.com/cookielab/traefik-middleware-request-logger"] = plugin{
		Create: pe796367f4fc3f245.CreateConfig,
		New:    pe796367f4fc3f245.New,
		Version: "v0.0.9",
	}

	pluginMap["github.com/corticph/queryparameter-to-header"] = plugin{
		Create: p326a4ccb2c20b464.CreateConfig,
		New:    p326a4ccb2c20b464.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/craigbrogle/traefik-s3-plugin"] = plugin{
		Create: p9e4ff126b442afca.CreateConfig,
		New:    p9e4ff126b442afca.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/crazygolem/traefik-subsonic-basicauth"] = plugin{
		Create: pd67d4e3d85be6872.CreateConfig,
		New:    pd67d4e3d85be6872.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/credibil/pluginauth"] = plugin{
		Create: pe6b629e19d6753b2.CreateConfig,
		New:    pe6b629e19d6753b2.New,
		Version: "v0.0.28",
	}

	pluginMap["github.com/csobrinho/traefik-plugin-s3-auth"] = plugin{
		Create: p3492cc9d89d970f.CreateConfig,
		New:    p3492cc9d89d970f.New,
		Version: "v0.0.14",
	}

	pluginMap["github.com/ctrl-hub/traefik-auditor"] = plugin{
		Create: p3a4054bd4c35afcc.CreateConfig,
		New:    p3a4054bd4c35afcc.New,
		Version: "v0.2.1",
	}

	pluginMap["github.com/Cubicroots-Playground/traefik-geoip-metrics-middleware"] = plugin{
		Create: pebf29ef77d346dd8.CreateConfig,
		New:    pebf29ef77d346dd8.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/CumpsD/edns0"] = plugin{
		Create: pc7dcc1dd7a095b44.CreateConfig,
		New:    pc7dcc1dd7a095b44.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/Cyb3r-Jak3/traefik-plugin-cloudflare"] = plugin{
		Create: p13d8d8cb7a7912a6.CreateConfig,
		New:    p13d8d8cb7a7912a6.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/danbiagini/traefik-cloud-saver"] = plugin{
		Create: paad0be7297c02941.CreateConfig,
		New:    paad0be7297c02941.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/danielbjornadal/traefik-cloudflare-plugin"] = plugin{
		Create: ped731d02e48f0d0.CreateConfig,
		New:    ped731d02e48f0d0.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/daniels0056/traefik-simpleredirect"] = plugin{
		Create: p493613a416227a78.CreateConfig,
		New:    p493613a416227a78.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/dararish/captcha-protect"] = plugin{
		Create: p8cbcffdc94924953.CreateConfig,
		New:    p8cbcffdc94924953.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/dariusandz/header-transmute"] = plugin{
		Create: p79a57243a729bee8.CreateConfig,
		New:    p79a57243a729bee8.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/darkweak/go-esi/middleware/traefik"] = plugin{
		Create: p30b60e7be766f883.CreateConfig,
		New:    p30b60e7be766f883.New,
		Version: "v0.0.6",
	}

	pluginMap["github.com/dashpool/dashmiddleware"] = plugin{
		Create: p54fd784626fb872f.CreateConfig,
		New:    p54fd784626fb872f.New,
		Version: "v0.0.35",
	}

	pluginMap["github.com/davewhit3/traefik-cf-device-detector"] = plugin{
		Create: p3dd9c389d70878a2.CreateConfig,
		New:    p3dd9c389d70878a2.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/david-garcia-garcia/traefik-geoblock"] = plugin{
		Create: pcdeb999149103e5f.CreateConfig,
		New:    pcdeb999149103e5f.New,
		Version: "v1.1.2-beta.2",
	}

	pluginMap["github.com/david-garcia-garcia/traefik-modsecurity"] = plugin{
		Create: p915df3171d91eeef.CreateConfig,
		New:    p915df3171d91eeef.New,
		Version: "v1.7.3",
	}

	pluginMap["github.com/david-garcia-garcia/traefik-realip"] = plugin{
		Create: pc2d49d1a932e5422.CreateConfig,
		New:    pc2d49d1a932e5422.New,
		Version: "v1.0.0-beta.3",
	}

	pluginMap["github.com/daxroc/traefik-jwt-org-redirect"] = plugin{
		Create: pc3edcbcd4adcba86.CreateConfig,
		New:    pc3edcbcd4adcba86.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/dcasia/plugin-cond-redirect"] = plugin{
		Create: pc2ba2ea112ba02b9.CreateConfig,
		New:    pc2ba2ea112ba02b9.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/dclairac/traefik-plugin-headers"] = plugin{
		Create: p8cbccd42d646cb31.CreateConfig,
		New:    p8cbccd42d646cb31.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/decodeex/traefik_middleware"] = plugin{
		Create: pbc2ffb98e055a33f.CreateConfig,
		New:    pbc2ffb98e055a33f.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/Desuuuu/traefik-cloudflare-plugin"] = plugin{
		Create: pc1ccefee96a8b186.CreateConfig,
		New:    pc1ccefee96a8b186.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/Desuuuu/traefik-real-ip-plugin"] = plugin{
		Create: pfec832e723e82f50.CreateConfig,
		New:    pfec832e723e82f50.New,
		Version: "v1.1.0",
	}

	pluginMap["github.com/dev-toolbox/traefik-plugin-parameters"] = plugin{
		Create: pc20f6ff9f42f104b.CreateConfig,
		New:    pc20f6ff9f42f104b.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/developmentaid-org/denyip"] = plugin{
		Create: p8d56739cef3fdf39.CreateConfig,
		New:    p8d56739cef3fdf39.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/dgzlopes/traefik-datadog-event"] = plugin{
		Create: p43fbefdb3b197eb1.CreateConfig,
		New:    p43fbefdb3b197eb1.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/dgzlopes/traefik-fault-injection"] = plugin{
		Create: pd012da2f6dbb4f3.CreateConfig,
		New:    pd012da2f6dbb4f3.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/DIE-Bonn/MatomoTracking"] = plugin{
		Create: pcbc8c3371b9d2983.CreateConfig,
		New:    pcbc8c3371b9d2983.New,
		Version: "v1.1.0",
	}

	pluginMap["github.com/Dimoniq/jwtvalidator"] = plugin{
		Create: pe8ad1623bb2bc858.CreateConfig,
		New:    pe8ad1623bb2bc858.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/dimorder/dimdanredirect"] = plugin{
		Create: p944c541f767590c3.CreateConfig,
		New:    p944c541f767590c3.New,
		Version: "v0.0.18",
	}

	pluginMap["github.com/DirtyCajunRice/subfilter"] = plugin{
		Create: p50325d2dc581fbc0.CreateConfig,
		New:    p50325d2dc581fbc0.New,
		Version: "v0.5.0",
	}

	pluginMap["github.com/discoverygarden/traefik-ultimate-bad-bot-blocker"] = plugin{
		Create: pfcb6a86df641353a.CreateConfig,
		New:    pfcb6a86df641353a.New,
		Version: "v0.2.1",
	}

	pluginMap["github.com/dkijkuit/azurejwttokenvalidation"] = plugin{
		Create: p7a623984ff72567b.CreateConfig,
		New:    p7a623984ff72567b.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/dkijkuit/checkheadersplugin"] = plugin{
		Create: p5bd7650ad3f43f35.CreateConfig,
		New:    p5bd7650ad3f43f35.New,
		Version: "v0.3.1",
	}

	pluginMap["github.com/dndll/header-to-queryparameter"] = plugin{
		Create: pe9c5ccd2679f882d.CreateConfig,
		New:    pe9c5ccd2679f882d.New,
		Version: "v1.0.0-rc2",
	}

	pluginMap["github.com/dobots/multiplexer-proxy"] = plugin{
		Create: p7048a7c357e8febc.CreateConfig,
		New:    p7048a7c357e8febc.New,
		Version: "v1.1.0",
	}

	pluginMap["github.com/DogAndHerDude/plugin-aheadinator"] = plugin{
		Create: p42746916597c3c76.CreateConfig,
		New:    p42746916597c3c76.New,
		Version: "v0.0.7",
	}

	pluginMap["github.com/domainesia/traefik-plugin-reformatheader"] = plugin{
		Create: pcd12a0ba08edf672.CreateConfig,
		New:    pcd12a0ba08edf672.New,
		Version: "v0.0.4-alpha.2",
	}

	pluginMap["github.com/dominion-solutions/traefik-filter-on-field"] = plugin{
		Create: p70be20fc50f50704.CreateConfig,
		New:    p70be20fc50f50704.New,
		Version: "v1.0.4",
	}

	pluginMap["github.com/dragosnutu/traefik-plugin"] = plugin{
		Create: pf36d0771cbb6b1bd.CreateConfig,
		New:    pf36d0771cbb6b1bd.New,
		Version: "v0.0.6",
	}

	pluginMap["github.com/dtomlinson91/traefik-api-key-middleware"] = plugin{
		Create: pcbf0c0286cfd3a59.CreateConfig,
		New:    pcbf0c0286cfd3a59.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/durvesh-palkar/traefik-epoch-header"] = plugin{
		Create: pf9af7d864fd28cde.CreateConfig,
		New:    pf9af7d864fd28cde.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/dzungmmp/host-header-plugin"] = plugin{
		Create: pa804270def4ac4c4.CreateConfig,
		New:    pa804270def4ac4c4.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/e-flux-platform/full-url-rewrite-traefik-plugin"] = plugin{
		Create: p7fc433ce188edf54.CreateConfig,
		New:    p7fc433ce188edf54.New,
		Version: "v0.0.7",
	}

	pluginMap["github.com/EasySolutionsIO/traefikxrequeststart"] = plugin{
		Create: pc218c96ea607897b.CreateConfig,
		New:    pc218c96ea607897b.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/ecov/traefik-plugin-introspect"] = plugin{
		Create: p2d01717153ed421b.CreateConfig,
		New:    p2d01717153ed421b.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/edelbluth/tm_http_redirect"] = plugin{
		Create: p84e52dbfb24c5649.CreateConfig,
		New:    p84e52dbfb24c5649.New,
		Version: "v0.2.2",
	}

	pluginMap["github.com/edelbluth/tm_no_ai_bots"] = plugin{
		Create: p753a9abaec686e97.CreateConfig,
		New:    p753a9abaec686e97.New,
		Version: "v0.2.4",
	}

	pluginMap["github.com/edgeflare/traefikopa"] = plugin{
		Create: p22430294162b32af.CreateConfig,
		New:    p22430294162b32af.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/edgeworx/static-response-plugin"] = plugin{
		Create: p1839341d301cce5b.CreateConfig,
		New:    p1839341d301cce5b.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/elee1766/traefik-gubernator-plugin"] = plugin{
		Create: pc8fcfabea557dac7.CreateConfig,
		New:    pc8fcfabea557dac7.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/ELLIO-Technology/ELLIO-Traefik-Middleware-Plugin"] = plugin{
		Create: p8e587c15275c044f.CreateConfig,
		New:    p8e587c15275c044f.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/emigrating/TraefikRealIPs"] = plugin{
		Create: pa9763b2bda45697d.CreateConfig,
		New:    pa9763b2bda45697d.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/esnunes/redirecterrors"] = plugin{
		Create: pd987e21b65277957.CreateConfig,
		New:    pd987e21b65277957.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/Evocelot/traefik-lazy-serve"] = plugin{
		Create: pbc1f8238e736a205.CreateConfig,
		New:    pbc1f8238e736a205.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/evolves-fr/traefik-plugin-redirect"] = plugin{
		Create: pa15254f6a22bce9a.CreateConfig,
		New:    pa15254f6a22bce9a.New,
		Version: "v1.2.0",
	}

	pluginMap["github.com/famedly/traefik-certauthz"] = plugin{
		Create: peeebecb8f427029d.CreateConfig,
		New:    peeebecb8f427029d.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/Farrukhraz/headersvalidator"] = plugin{
		Create: pe3b8cafeddcc3d27.CreateConfig,
		New:    pe3b8cafeddcc3d27.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/Farrukhraz/jwt2headers"] = plugin{
		Create: p30750a8907c53787.CreateConfig,
		New:    p30750a8907c53787.New,
		Version: "v0.2.1",
	}

	pluginMap["github.com/fdufault/mapheaders"] = plugin{
		Create: pcd75ee182eb8e017.CreateConfig,
		New:    pcd75ee182eb8e017.New,
		Version: "v0.0.9",
	}

	pluginMap["github.com/feedzai/traefikRequestTimestamps"] = plugin{
		Create: p8b357f59d93a8b94.CreateConfig,
		New:    p8b357f59d93a8b94.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/Felioh/traefik-post-to-delete"] = plugin{
		Create: p57af4bb55a21399c.CreateConfig,
		New:    p57af4bb55a21399c.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/filipmiik/traefik-standard-proxy-headers"] = plugin{
		Create: p2fabdf477ff1fb70.CreateConfig,
		New:    p2fabdf477ff1fb70.New,
		Version: "v1.0.8",
	}

	pluginMap["github.com/FinalCAD/TraefikRegionalPlugin"] = plugin{
		Create: pcf8525f4f548fe30.CreateConfig,
		New:    pcf8525f4f548fe30.New,
		Version: "v0.0.10",
	}

	pluginMap["github.com/firstred/crowdsec-bouncer-traefik-plugin"] = plugin{
		Create: pd50e7e1bc9a14b38.CreateConfig,
		New:    pd50e7e1bc9a14b38.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/fliot/fail2ban"] = plugin{
		Create: pc75a72fe73be7b7c.CreateConfig,
		New:    pc75a72fe73be7b7c.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/florentinl/websitepathconverter"] = plugin{
		Create: pcb4a2e6741ccbf78.CreateConfig,
		New:    pcb4a2e6741ccbf78.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/FlowingSPDG/traefik-plugin-pick-response-json"] = plugin{
		Create: p54286df88f888b55.CreateConfig,
		New:    p54286df88f888b55.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/FlowingSPDG/traefik-plugin-query-to-json"] = plugin{
		Create: pfb85d5c37ae33523.CreateConfig,
		New:    pfb85d5c37ae33523.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/fma965/cloudflarewarp"] = plugin{
		Create: p3be782b5ca5cc675.CreateConfig,
		New:    p3be782b5ca5cc675.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/fopina/traefik-commonname-validator-plugin"] = plugin{
		Create: p64764d5564df6c2d.CreateConfig,
		New:    p64764d5564df6c2d.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/formancehq/gateway-plugin-auth"] = plugin{
		Create: p39fdb68b2c2c032f.CreateConfig,
		New:    p39fdb68b2c2c032f.New,
		Version: "v0.1.17",
	}

	pluginMap["github.com/fosrl/badger"] = plugin{
		Create: p9add789cae89267d.CreateConfig,
		New:    p9add789cae89267d.New,
		Version: "v1.2.0",
	}

	pluginMap["github.com/FoxxMD/traefik-rybbit-feeder"] = plugin{
		Create: pcee99c7126bf02dc.CreateConfig,
		New:    pcee99c7126bf02dc.New,
		Version: "v0.13.4",
	}

	pluginMap["github.com/frankforpresident/traefik-plugin-validate-headers"] = plugin{
		Create: p94a67f7cfb4597f3.CreateConfig,
		New:    p94a67f7cfb4597f3.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/fzoli/traefiklogger"] = plugin{
		Create: pacc4f657fc50a24.CreateConfig,
		New:    pacc4f657fc50a24.New,
		Version: "v0.11.5",
	}

	pluginMap["github.com/gaborini/traefik-header-rename-plugin"] = plugin{
		Create: pa232bc2f8777e8b0.CreateConfig,
		New:    pa232bc2f8777e8b0.New,
		Version: "v1.0.4",
	}

	pluginMap["github.com/galaxias/traefik-vault-auth"] = plugin{
		Create: p33aec9e7e8ddd85a.CreateConfig,
		New:    p33aec9e7e8ddd85a.New,
		Version: "v0.2.3",
	}

	pluginMap["github.com/georg-jung/github-webhook-middleware"] = plugin{
		Create: peaa48e445796cf49.CreateConfig,
		New:    peaa48e445796cf49.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/germangorodnev/multi-jwt-validation-middleware"] = plugin{
		Create: pcef173d3ad7a6d1f.CreateConfig,
		New:    pcef173d3ad7a6d1f.New,
		Version: "v0.3.1",
	}

	pluginMap["github.com/ghnexpress/traefik-cache"] = plugin{
		Create: p77739c7a1b7ea45c.CreateConfig,
		New:    p77739c7a1b7ea45c.New,
		Version: "v0.0.9",
	}

	pluginMap["github.com/ghnexpress/traefik-ratelimit"] = plugin{
		Create: pea9acb1c37d051cc.CreateConfig,
		New:    pea9acb1c37d051cc.New,
		Version: "v0.0.20",
	}

	pluginMap["github.com/giacomoferretti/add-missing-headers"] = plugin{
		Create: p6f0bae79b43254bd.CreateConfig,
		New:    p6f0bae79b43254bd.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/GiGInnovationLabs/traefikgeoip2"] = plugin{
		Create: p2f76229b2b19466e.CreateConfig,
		New:    p2f76229b2b19466e.New,
		Version: "v0.20.1",
	}

	pluginMap["github.com/GiGInnovationLabs/traefikuseragent"] = plugin{
		Create: pd551b66433983b50.CreateConfig,
		New:    pd551b66433983b50.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/glefer/sensitive-files-blocker"] = plugin{
		Create: p7dc8234e42a6f197.CreateConfig,
		New:    p7dc8234e42a6f197.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/GLMONTER/crlchecker"] = plugin{
		Create: pd30b6ca553efb2ef.CreateConfig,
		New:    pd30b6ca553efb2ef.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/golubev-ml/white-elephant"] = plugin{
		Create: p2b02da3ef7c83ba8.CreateConfig,
		New:    p2b02da3ef7c83ba8.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/gpabijan88/traefik-auth-middleware"] = plugin{
		Create: p13ec94e5ac60108a.CreateConfig,
		New:    p13ec94e5ac60108a.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/Gwojda/keycloakopenid"] = plugin{
		Create: p20f3e33ce2ef1871.CreateConfig,
		New:    p20f3e33ce2ef1871.New,
		Version: "v0.1.36",
	}

	pluginMap["github.com/h3xsh/traefik-middleware"] = plugin{
		Create: pdf4be6c472785b45.CreateConfig,
		New:    pdf4be6c472785b45.New,
		Version: "v0.3.0",
	}

	pluginMap["github.com/hajimAIM/badger"] = plugin{
		Create: peda8636352895044.CreateConfig,
		New:    peda8636352895044.New,
		Version: "v1.2.1",
	}

	pluginMap["github.com/helpshift/timestamp-injector"] = plugin{
		Create: p2ce8d1268923496.CreateConfig,
		New:    p2ce8d1268923496.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/hhftechnology/bandwidthlimiter"] = plugin{
		Create: p982e34b8d9ee5cd8.CreateConfig,
		New:    p982e34b8d9ee5cd8.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/hhftechnology/ipwhitelistshaper"] = plugin{
		Create: pfa7d0692f35dfdaf.CreateConfig,
		New:    pfa7d0692f35dfdaf.New,
		Version: "v1.0.8",
	}

	pluginMap["github.com/hhftechnology/statiq"] = plugin{
		Create: p13965f73a02236f.CreateConfig,
		New:    p13965f73a02236f.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/hhftechnology/tailscale-access"] = plugin{
		Create: p58395cca998bb74c.CreateConfig,
		New:    p58395cca998bb74c.New,
		Version: "v2.0.0",
	}

	pluginMap["github.com/hhftechnology/traefik-queue-manager"] = plugin{
		Create: p3e27f878d2ba5b60.CreateConfig,
		New:    p3e27f878d2ba5b60.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/hiasr/forwardmiddleware"] = plugin{
		Create: p891f9ecdeb407f2.CreateConfig,
		New:    p891f9ecdeb407f2.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/hiasr/jwtfieldsheader"] = plugin{
		Create: p44688d2249d7989c.CreateConfig,
		New:    p44688d2249d7989c.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/holysoles/bot-wrangler-traefik-plugin"] = plugin{
		Create: p6c0539f8e449a394.CreateConfig,
		New:    p6c0539f8e449a394.New,
		Version: "v0.6.0",
	}

	pluginMap["github.com/Hongbo-Miao/traefik-plugin-disable-graphql-introspection"] = plugin{
		Create: pfa04a697ab1098e6.CreateConfig,
		New:    pfa04a697ab1098e6.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/honghainguyen777/traefik-modsecurity-plugin"] = plugin{
		Create: p9a6f63b2bc604777.CreateConfig,
		New:    p9a6f63b2bc604777.New,
		Version: "v1.6.7",
	}

	pluginMap["github.com/hukumonline-com/traefik-modifier-plugin"] = plugin{
		Create: p62e73c0a9f22b92.CreateConfig,
		New:    p62e73c0a9f22b92.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/Hvid/proxyHeaders"] = plugin{
		Create: p9162e5b5a57cbadf.CreateConfig,
		New:    p9162e5b5a57cbadf.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/hwmland/dbgdump"] = plugin{
		Create: paaf2f67559976b34.CreateConfig,
		New:    paaf2f67559976b34.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/iamolegga/plugin-simplecache"] = plugin{
		Create: pe9094b35cb141947.CreateConfig,
		New:    pe9094b35cb141947.New,
		Version: "v0.4.0",
	}

	pluginMap["github.com/igoooor/conteo-traefik-cache"] = plugin{
		Create: pce823b8b679da645.CreateConfig,
		New:    pce823b8b679da645.New,
		Version: "v1.1.0",
	}

	pluginMap["github.com/igoooor/plugin-simplecache-conteo"] = plugin{
		Create: p219f1b52747c8773.CreateConfig,
		New:    p219f1b52747c8773.New,
		Version: "v1.0.8",
	}

	pluginMap["github.com/ijo42/corsmiddleware"] = plugin{
		Create: p571c122121196101.CreateConfig,
		New:    p571c122121196101.New,
		Version: "v0.1.4",
	}

	pluginMap["github.com/iloahz/traefik-plugin-manual-access-control/plugin"] = plugin{
		Create: paec38f5a7bb893c8.CreateConfig,
		New:    paec38f5a7bb893c8.New,
		Version: "v0.1.9",
	}

	pluginMap["github.com/im-kulikov/traefik-provider"] = plugin{
		Create: pb0c92171b6df5629.CreateConfig,
		New:    pb0c92171b6df5629.New,
		Version: "v0.3.0-rc.1",
	}

	pluginMap["github.com/Imaskiller/ddns-allowlist"] = plugin{
		Create: peea659e6e4be8b63.CreateConfig,
		New:    peea659e6e4be8b63.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/imKota/traefik-maintenance-warden"] = plugin{
		Create: pd0db4d20eef66f05.CreateConfig,
		New:    pd0db4d20eef66f05.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/imKota/traefik-maintenance"] = plugin{
		Create: p3e8f0cbedb8b0f7e.CreateConfig,
		New:    p3e8f0cbedb8b0f7e.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/inalbilal/traefik-cookie-auth"] = plugin{
		Create: p9735a08c92f9657c.CreateConfig,
		New:    p9735a08c92f9657c.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/indivisible/redirecterrors"] = plugin{
		Create: pdc7e0b4976f5bd23.CreateConfig,
		New:    pdc7e0b4976f5bd23.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/init-object/redirect-ipv6"] = plugin{
		Create: pa1e30d8a617f6d43.CreateConfig,
		New:    pa1e30d8a617f6d43.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/inventi/traefik-header-transmute"] = plugin{
		Create: p23a012e62d4329f5.CreateConfig,
		New:    p23a012e62d4329f5.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/invit/traefik-cloudfront"] = plugin{
		Create: p18cd20703325be7a.CreateConfig,
		New:    p18cd20703325be7a.New,
		Version: "v0.0.6",
	}

	pluginMap["github.com/iolabs-ag/traefik-throttle"] = plugin{
		Create: p4da3e3d0e2d9f8cc.CreateConfig,
		New:    p4da3e3d0e2d9f8cc.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/ion-toolbox/traefik-jwt-eddsa"] = plugin{
		Create: p35d02e9f39896fa1.CreateConfig,
		New:    p35d02e9f39896fa1.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/irotem/jwt-rewrite"] = plugin{
		Create: p9149cd0d96c8898c.CreateConfig,
		New:    p9149cd0d96c8898c.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/isotes/traefik-outgoing-oauth2-cc"] = plugin{
		Create: p266df9345eb8c41c.CreateConfig,
		New:    p266df9345eb8c41c.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/itninja04/traefik-gelf-plugin"] = plugin{
		Create: pd681a5a74f269fcc.CreateConfig,
		New:    pd681a5a74f269fcc.New,
		Version: "v0.1.91",
	}

	pluginMap["github.com/iYUYUE/traefik-forwarded-real-ip"] = plugin{
		Create: p76884fad388d888.CreateConfig,
		New:    p76884fad388d888.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/J4NS-R/traefik-oauth-upstream"] = plugin{
		Create: pf8660777e2f7b671.CreateConfig,
		New:    pf8660777e2f7b671.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/jacekstarondiscovery/traefik-redirector"] = plugin{
		Create: p47d0b143dc82232a.CreateConfig,
		New:    p47d0b143dc82232a.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/jackhillman/pluginesi"] = plugin{
		Create: p3c43d388dec7b577.CreateConfig,
		New:    p3c43d388dec7b577.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/Jakob3xD/checkheaders"] = plugin{
		Create: pf33fdfbabe9207ec.CreateConfig,
		New:    pf33fdfbabe9207ec.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/jamesmcroft/traefik-plugin-return-response"] = plugin{
		Create: p73cfb2628ab7a001.CreateConfig,
		New:    p73cfb2628ab7a001.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/jamesmcroft/traefik-plugin-rewrite-response-headers"] = plugin{
		Create: p8a090d24f5aa6024.CreateConfig,
		New:    p8a090d24f5aa6024.New,
		Version: "v1.1.2",
	}

	pluginMap["github.com/jaybubs/headerdump"] = plugin{
		Create: p26fee8d065b0c774.CreateConfig,
		New:    p26fee8d065b0c774.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/jdel/staticresponse"] = plugin{
		Create: pf0249d0ea5b742d6.CreateConfig,
		New:    pf0249d0ea5b742d6.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/jeessy2/traefik-ip2region"] = plugin{
		Create: p3546926426952a37.CreateConfig,
		New:    p3546926426952a37.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/jeppestaerk/traefik-xff-to-xrealip"] = plugin{
		Create: pf7964fc3d99c5064.CreateConfig,
		New:    pf7964fc3d99c5064.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/jerrywoo96/AddForwardedHeader"] = plugin{
		Create: p2894b10cf8a3ebcb.CreateConfig,
		New:    p2894b10cf8a3ebcb.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/jerrywoo96/AddMissingHeaders"] = plugin{
		Create: pb539dcbbcc20012.CreateConfig,
		New:    pb539dcbbcc20012.New,
		Version: "v1.1.2",
	}

	pluginMap["github.com/jetersen/traefik-cloudfront-xforwarded"] = plugin{
		Create: pf16b169502d7709e.CreateConfig,
		New:    pf16b169502d7709e.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/jghaanstra/badger"] = plugin{
		Create: pcd4907694dda8483.CreateConfig,
		New:    pcd4907694dda8483.New,
		Version: "v1.2.3",
	}

	pluginMap["github.com/jiangwennn/traefik-plugin-ip2location-redirect"] = plugin{
		Create: p2ccaa8c271c0ad4a.CreateConfig,
		New:    p2ccaa8c271c0ad4a.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/JimCronqvist/traefik-api-key-auth"] = plugin{
		Create: pa45ad1924fb8b0d0.CreateConfig,
		New:    pa45ad1924fb8b0d0.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/JimmyTsai16/cloudflarewarp"] = plugin{
		Create: pc0d717edfc425321.CreateConfig,
		New:    pc0d717edfc425321.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/JimmyTsai16/MethodAllowed"] = plugin{
		Create: pba597edd138da595.CreateConfig,
		New:    pba597edd138da595.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/jmcarbo/keycloakopenid"] = plugin{
		Create: p1a7cfbb11afe3d58.CreateConfig,
		New:    p1a7cfbb11afe3d58.New,
		Version: "v0.1.40",
	}

	pluginMap["github.com/joegarb/traefik-throttle"] = plugin{
		Create: p624c50f1ce964cab.CreateConfig,
		New:    p624c50f1ce964cab.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/joinrepublic/traefik-csp-middleware"] = plugin{
		Create: p9c0e0c7786e5877d.CreateConfig,
		New:    p9c0e0c7786e5877d.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/JonasSchubert/traefik-allow-countries"] = plugin{
		Create: p1780b1f32b3cc296.CreateConfig,
		New:    p1780b1f32b3cc296.New,
		Version: "v1.2.0",
	}

	pluginMap["github.com/JonasSchubert/traefik-block-paths"] = plugin{
		Create: p29813527b2f32857.CreateConfig,
		New:    p29813527b2f32857.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/jonathaanhs/traefik-forward-auth-body"] = plugin{
		Create: pfc77208eb17b58e2.CreateConfig,
		New:    pfc77208eb17b58e2.New,
		Version: "v2.0.4",
	}

	pluginMap["github.com/JoshuaBowerman/TraefikCookiePathReplacement"] = plugin{
		Create: p15b21f8faa6921da.CreateConfig,
		New:    p15b21f8faa6921da.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/joy2fun/traefik-plugin-log-request"] = plugin{
		Create: p330cd97b7796f89e.CreateConfig,
		New:    p330cd97b7796f89e.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/jpxd/torblock"] = plugin{
		Create: pca9e90a96a2f87e7.CreateConfig,
		New:    pca9e90a96a2f87e7.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/jramsgz/traefik-real-ip"] = plugin{
		Create: p961124afb2837b51.CreateConfig,
		New:    p961124afb2837b51.New,
		Version: "v1.0.6",
	}

	pluginMap["github.com/jrg1381/smrequestid"] = plugin{
		Create: p7e6a1f5e51ac9eb0.CreateConfig,
		New:    p7e6a1f5e51ac9eb0.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/Ju0x/traefik-security-txt"] = plugin{
		Create: p72ef0dd1e0dc89df.CreateConfig,
		New:    p72ef0dd1e0dc89df.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/juitde/traefik-plugin-fail2ban"] = plugin{
		Create: pcec35f4f564a11bb.CreateConfig,
		New:    pcec35f4f564a11bb.New,
		Version: "v0.3.0",
	}

	pluginMap["github.com/K8Trust/authcookie"] = plugin{
		Create: p76c5b6318bc9bb31.CreateConfig,
		New:    p76c5b6318bc9bb31.New,
		Version: "v1.0.6",
	}

	pluginMap["github.com/K8Trust/traefikwsbalancer"] = plugin{
		Create: p45cf6dafefaa00c8.CreateConfig,
		New:    p45cf6dafefaa00c8.New,
		Version: "v1.0.58",
	}

	pluginMap["github.com/kahf-infra/traefikPluginPathHeader"] = plugin{
		Create: pcd4264c7d7361a5c.CreateConfig,
		New:    pcd4264c7d7361a5c.New,
		Version: "v0.1.6",
	}

	pluginMap["github.com/kaitencloud/traefik-svix-plugin"] = plugin{
		Create: pcadd02ceba82a753.CreateConfig,
		New:    pcadd02ceba82a753.New,
		Version: "v0.0.6",
	}

	pluginMap["github.com/KaloyanYosifov/traefik-plugin-insert-custom-header"] = plugin{
		Create: p391899cdc83d6c23.CreateConfig,
		New:    p391899cdc83d6c23.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/kav789/plugindemo"] = plugin{
		Create: p59069a2867e3a69e.CreateConfig,
		New:    p59069a2867e3a69e.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/Kazyini/yatmp"] = plugin{
		Create: pcf642a462056a768.CreateConfig,
		New:    pcf642a462056a768.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/kevtainer/denyip"] = plugin{
		Create: p3e99f1e622944066.CreateConfig,
		New:    p3e99f1e622944066.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/killer-djon/traefik-correlation"] = plugin{
		Create: pf62417f6800f1d24.CreateConfig,
		New:    pf62417f6800f1d24.New,
		Version: "v1.3.1",
	}

	pluginMap["github.com/killer-djon/traefik-http2amqp"] = plugin{
		Create: pa9ef56fa8cdf2289.CreateConfig,
		New:    pa9ef56fa8cdf2289.New,
		Version: "v1.4.2",
	}

	pluginMap["github.com/kingjan1999/traefik-plugin-custom-mtls"] = plugin{
		Create: pacc32800cbce2997.CreateConfig,
		New:    pacc32800cbce2997.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/kingjan1999/traefik-plugin-exception-authbasic"] = plugin{
		Create: pe08462bddb97fe47.CreateConfig,
		New:    pe08462bddb97fe47.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/kingjan1999/traefik-plugin-query-modification"] = plugin{
		Create: p47cbc01a8d63abc6.CreateConfig,
		New:    p47cbc01a8d63abc6.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/KizzyCode/mtlsrules-traefik-golang"] = plugin{
		Create: pa4b4a6befb6268a8.CreateConfig,
		New:    pa4b4a6befb6268a8.New,
		Version: "v0.2.2",
	}

	pluginMap["github.com/Knight-7/addheader"] = plugin{
		Create: p9c16a7b41332f31a.CreateConfig,
		New:    p9c16a7b41332f31a.New,
		Version: "v0.0.8",
	}

	pluginMap["github.com/KodeyThomas/traefik-oidc"] = plugin{
		Create: pf3d331d2fe105e89.CreateConfig,
		New:    pf3d331d2fe105e89.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/Koellewe/traefik-oauth-upstream"] = plugin{
		Create: p988a639bb29f2bc6.CreateConfig,
		New:    p988a639bb29f2bc6.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/korteke/traefik-waiting-room"] = plugin{
		Create: p470214870f6c7858.CreateConfig,
		New:    p470214870f6c7858.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/kotalco/crossover-activity"] = plugin{
		Create: p907527345c6867e7.CreateConfig,
		New:    p907527345c6867e7.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/kotalco/crossover-blacklist"] = plugin{
		Create: pc1a02744e609d62.CreateConfig,
		New:    pc1a02744e609d62.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/kotalco/crossover-cache"] = plugin{
		Create: pee23a928ee8f3089.CreateConfig,
		New:    pee23a928ee8f3089.New,
		Version: "v1.1.0",
	}

	pluginMap["github.com/kotalco/crossover-limiter"] = plugin{
		Create: pa8841e60f65adcee.CreateConfig,
		New:    pa8841e60f65adcee.New,
		Version: "v0.1.8",
	}

	pluginMap["github.com/kotalco/crossover-managed"] = plugin{
		Create: p64526958e05912b0.CreateConfig,
		New:    p64526958e05912b0.New,
		Version: "v0.2.0-rc",
	}

	pluginMap["github.com/kotalco/crossover"] = plugin{
		Create: p366480a0d009070f.CreateConfig,
		New:    p366480a0d009070f.New,
		Version: "v0.1.8",
	}

	pluginMap["github.com/krokodaws/traefik-hidden-auth"] = plugin{
		Create: p6d6f857588df89cf.CreateConfig,
		New:    p6d6f857588df89cf.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/kucjac/traefik-block-ua"] = plugin{
		Create: p1cd8a0da46eb74ce.CreateConfig,
		New:    p1cd8a0da46eb74ce.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/kucjac/traefik-plugin-geoblock"] = plugin{
		Create: p858af0290f4b17b5.CreateConfig,
		New:    p858af0290f4b17b5.New,
		Version: "v0.2.4",
	}

	pluginMap["github.com/kumina/headers-by-request"] = plugin{
		Create: pf791b567886bdd99.CreateConfig,
		New:    pf791b567886bdd99.New,
		Version: "v0.0.10",
	}

	pluginMap["github.com/kumina/traefik-routing-plugin"] = plugin{
		Create: pd2cb2aaa3f9ebfc7.CreateConfig,
		New:    pd2cb2aaa3f9ebfc7.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/kuzzleio/traefik-header-transform"] = plugin{
		Create: pdf580154b6563075.CreateConfig,
		New:    pdf580154b6563075.New,
		Version: "v0.1.18",
	}

	pluginMap["github.com/kvncrw/denyip"] = plugin{
		Create: p929f2d61313fb955.CreateConfig,
		New:    p929f2d61313fb955.New,
		Version: "v2.0.0",
	}

	pluginMap["github.com/kyaxcorp/traefikdisolver"] = plugin{
		Create: pe41aea04f9d0311e.CreateConfig,
		New:    pe41aea04f9d0311e.New,
		Version: "v1.0.9",
	}

	pluginMap["github.com/kzmake/traefik-plugin-forward-request"] = plugin{
		Create: pd3ce1205f6d69179.CreateConfig,
		New:    pd3ce1205f6d69179.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/l4rm4nd/traefik-warp"] = plugin{
		Create: p434f677f11c2cd76.CreateConfig,
		New:    p434f677f11c2cd76.New,
		Version: "v1.1.5",
	}

	pluginMap["github.com/Lambda-IT/traefik-plugin-cookie-flags"] = plugin{
		Create: p792336c8a3eef78b.CreateConfig,
		New:    p792336c8a3eef78b.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/LASER-Yi/traefik-drop-connection"] = plugin{
		Create: pb53439c689c23e9e.CreateConfig,
		New:    pb53439c689c23e9e.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/LaurenceJJones/password-protect-traefik-plugin"] = plugin{
		Create: p4ab9f1c41e2fc96d.CreateConfig,
		New:    p4ab9f1c41e2fc96d.New,
		Version: "v0.0.6",
	}

	pluginMap["github.com/lazzio/cleanclientip"] = plugin{
		Create: pe7791fd7814826ce.CreateConfig,
		New:    pe7791fd7814826ce.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/leesalminen/kratos-session-extend"] = plugin{
		Create: p6fe7848921f297f2.CreateConfig,
		New:    p6fe7848921f297f2.New,
		Version: "v0.4.0",
	}

	pluginMap["github.com/legege/jwt-validation-middleware"] = plugin{
		Create: p79ee65b399e9297f.CreateConfig,
		New:    p79ee65b399e9297f.New,
		Version: "v0.2.1",
	}

	pluginMap["github.com/leonjza/trauth"] = plugin{
		Create: p9f2af4bafd61e63c.CreateConfig,
		New:    p9f2af4bafd61e63c.New,
		Version: "v1.6.7",
	}

	pluginMap["github.com/Lepkem/traefik-plugin-response-code-override"] = plugin{
		Create: p13253ec47513795.CreateConfig,
		New:    p13253ec47513795.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/LeslieRan/pluginDemo"] = plugin{
		Create: pf8fa4936eba58097.CreateConfig,
		New:    pf8fa4936eba58097.New,
		Version: "v0.3",
	}

	pluginMap["github.com/libops/captcha-protect"] = plugin{
		Create: p908773e7f75204f4.CreateConfig,
		New:    p908773e7f75204f4.New,
		Version: "v1.10.0",
	}

	pluginMap["github.com/lifter-ai/auth-token-exchange-plugin"] = plugin{
		Create: p716470db4ab7c39a.CreateConfig,
		New:    p716470db4ab7c39a.New,
		Version: "v0.1.9",
	}

	pluginMap["github.com/Lightirius/traefik-auth-converter"] = plugin{
		Create: p26f636bd0a87ece5.CreateConfig,
		New:    p26f636bd0a87ece5.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/lion7/traefik-jwt-headers-plugin"] = plugin{
		Create: pcbe145b97ebcf256.CreateConfig,
		New:    pcbe145b97ebcf256.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/LiquidLogicLabs/traefik-plugin-cors-regex"] = plugin{
		Create: pb7f95f2745facfac.CreateConfig,
		New:    pb7f95f2745facfac.New,
		Version: "v0.1.8",
	}

	pluginMap["github.com/livekit/traefik-readiness-plugin"] = plugin{
		Create: pdc7058c43c0e7e19.CreateConfig,
		New:    pdc7058c43c0e7e19.New,
		Version: "v0.0.2-beta.2",
	}

	pluginMap["github.com/LiveOakLabs/traefik_middleware_sigv4"] = plugin{
		Create: pf7ad488f882ea159.CreateConfig,
		New:    pf7ad488f882ea159.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/lixianliang/traefik-plugin-return-response"] = plugin{
		Create: pea45fa87b1248d55.CreateConfig,
		New:    pea45fa87b1248d55.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/lleadbet/traefik-plugin-cache-by-route"] = plugin{
		Create: pb941033aad5d2a62.CreateConfig,
		New:    pb941033aad5d2a62.New,
		Version: "v0.0.7",
	}

	pluginMap["github.com/longbridgeapp/traefik-session-max-age"] = plugin{
		Create: pc9eb247d85877a4.CreateConfig,
		New:    pc9eb247d85877a4.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/louiscavalcante/easy-traefik-rate-limit-jwt"] = plugin{
		Create: pf4741024bda5ad60.CreateConfig,
		New:    pf4741024bda5ad60.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/lsz66/traefik-cas-plugin"] = plugin{
		Create: pc040f583cfec259.CreateConfig,
		New:    pc040f583cfec259.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/luizfonseca/traefik-github-oauth-plugin"] = plugin{
		Create: p7f8c5512c9136a5d.CreateConfig,
		New:    p7f8c5512c9136a5d.New,
		Version: "v0.7.0",
	}

	pluginMap["github.com/lukas-r/traefik-subdomain-path-rewrite-plugin"] = plugin{
		Create: pd230459b0f4919c6.CreateConfig,
		New:    pd230459b0f4919c6.New,
		Version: "v0.3.1",
	}

	pluginMap["github.com/lukaszraczylo/traefikoidc"] = plugin{
		Create: pb081345803255220.CreateConfig,
		New:    pb081345803255220.New,
		Version: "v0.7.10",
	}

	pluginMap["github.com/LumensGroup/scanblock"] = plugin{
		Create: pa0274d7d826c58ce.CreateConfig,
		New:    pa0274d7d826c58ce.New,
		Version: "v1.1.0",
	}

	pluginMap["github.com/luthfikw/traefik-real-ip-go"] = plugin{
		Create: pce921b3d2e48dd97.CreateConfig,
		New:    pce921b3d2e48dd97.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/LyuHe-uestc/traefik-plugin-check"] = plugin{
		Create: p796763078307d7b.CreateConfig,
		New:    p796763078307d7b.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/LyuHe-uestc/traefik-plugin-ipblacklist"] = plugin{
		Create: p3627fd471bb8bce8.CreateConfig,
		New:    p3627fd471bb8bce8.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/LyuHe-uestc/traefik-plugin-token-auth"] = plugin{
		Create: pcff558478f73050e.CreateConfig,
		New:    pcff558478f73050e.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/m-riedel/traefik-plugin-redirect-on-status"] = plugin{
		Create: pe4d496d156f52878.CreateConfig,
		New:    pe4d496d156f52878.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/m4rc3l-h3/headermodifications"] = plugin{
		Create: pc305703915052bda.CreateConfig,
		New:    pc305703915052bda.New,
		Version: "v0.1.5",
	}

	pluginMap["github.com/MadddinTribleD/traefikaggregator"] = plugin{
		Create: p6685fcd74b181edf.CreateConfig,
		New:    p6685fcd74b181edf.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/madebymode/traefik-modsecurity-plugin"] = plugin{
		Create: p1e0ca064094e43a.CreateConfig,
		New:    p1e0ca064094e43a.New,
		Version: "v1.6.0",
	}

	pluginMap["github.com/maiaaraujo5/traefik-jwt-claims"] = plugin{
		Create: p62c90b618d41ceb.CreateConfig,
		New:    p62c90b618d41ceb.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/MarkusJx/traefik-wol"] = plugin{
		Create: pf9f5d12596e4e3ff.CreateConfig,
		New:    pf9f5d12596e4e3ff.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/Maronato/traefik_geoip"] = plugin{
		Create: p76628e3e1cef8950.CreateConfig,
		New:    p76628e3e1cef8950.New,
		Version: "v0.5.0",
	}

	pluginMap["github.com/maxlerebourg/crowdsec-bouncer-traefik-plugin"] = plugin{
		Create: p64ca573d55be393e.CreateConfig,
		New:    p64ca573d55be393e.New,
		Version: "v1.4.5",
	}

	pluginMap["github.com/mayankkumar2/traefik-plugin-securum-exire"] = plugin{
		Create: p5cae4ccfe2fe5c43.CreateConfig,
		New:    p5cae4ccfe2fe5c43.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/mdklapwijk/traefik-plugin-request-id"] = plugin{
		Create: p8fffaddba7e6e481.CreateConfig,
		New:    p8fffaddba7e6e481.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/mdouchement/geoblock"] = plugin{
		Create: p67a35149ba3c0acc.CreateConfig,
		New:    p67a35149ba3c0acc.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/Medad-ai/medad-jwt-middleware"] = plugin{
		Create: pad6d656de6677dca.CreateConfig,
		New:    pad6d656de6677dca.New,
		Version: "v1.0",
	}

	pluginMap["github.com/Medzoner/traefik-plugin-cors-preflight"] = plugin{
		Create: p1d5d1c74b4b2775b.CreateConfig,
		New:    p1d5d1c74b4b2775b.New,
		Version: "v1.0.7",
	}

	pluginMap["github.com/melchor629/traefik-error-page"] = plugin{
		Create: pab026d61ddf42814.CreateConfig,
		New:    pab026d61ddf42814.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/Menfre01/conditionheader"] = plugin{
		Create: p31df7aecf93b1802.CreateConfig,
		New:    p31df7aecf93b1802.New,
		Version: "v0.0.7",
	}

	pluginMap["github.com/MGDIS/traefik-apimanager-plugin"] = plugin{
		Create: p5a52434be1fb502.CreateConfig,
		New:    p5a52434be1fb502.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/Miladbr/tlscrlchecker"] = plugin{
		Create: pef095dee062153ab.CreateConfig,
		New:    pef095dee062153ab.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/milosdjurdjevic/traefik-deep-linking-middleware"] = plugin{
		Create: pb9398ef1e2bf64e3.CreateConfig,
		New:    pb9398ef1e2bf64e3.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/Miromani4/traefik-plugin-AdminAPI_WebUI"] = plugin{
		Create: p2421326a87ae9f6b.CreateConfig,
		New:    p2421326a87ae9f6b.New,
		Version: "v1.3.1",
	}

	pluginMap["github.com/mlambda-net/secret-header"] = plugin{
		Create: pd89a44e9beb72b7b.CreateConfig,
		New:    pd89a44e9beb72b7b.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/mmpx12/traefik-secpath"] = plugin{
		Create: pb07e8d796f01b403.CreateConfig,
		New:    pb07e8d796f01b403.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/mohamed-abdelrhman/traefikloggerbridge"] = plugin{
		Create: p5d091ad606195c6b.CreateConfig,
		New:    p5d091ad606195c6b.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/momayyez/authztraefikgateway"] = plugin{
		Create: pf27ff1bc1dbc0e2.CreateConfig,
		New:    pf27ff1bc1dbc0e2.New,
		Version: "v2.0.2",
	}

	pluginMap["github.com/momayyez/traefikauthz"] = plugin{
		Create: pe20a41c12f2a5bc.CreateConfig,
		New:    pe20a41c12f2a5bc.New,
		Version: "v0.1.5",
	}

	pluginMap["github.com/moonlight8978/traefik-cloudflare-geoblock"] = plugin{
		Create: p4a3d7424fa61bb0f.CreateConfig,
		New:    p4a3d7424fa61bb0f.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/moonlightwatch/MethodBlock"] = plugin{
		Create: p3f06b657559bee4c.CreateConfig,
		New:    p3f06b657559bee4c.New,
		Version: "v0.1.4",
	}

	pluginMap["github.com/moonlightwatch/referer"] = plugin{
		Create: p22180835c1c47639.CreateConfig,
		New:    p22180835c1c47639.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/moonlightwatch/ReturnClientIP"] = plugin{
		Create: pddbb97c9e007955.CreateConfig,
		New:    pddbb97c9e007955.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/Morozzzko/traefik-csp-middleware"] = plugin{
		Create: p984593cfac712905.CreateConfig,
		New:    p984593cfac712905.New,
		Version: "v2.3.4",
	}

	pluginMap["github.com/mrambossek/traefik-extraheaders"] = plugin{
		Create: p48a540bf8b8025bb.CreateConfig,
		New:    p48a540bf8b8025bb.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/mrdrelar/traefik-plugin-rewriteheader"] = plugin{
		Create: pcb9612c6ed0c518b.CreateConfig,
		New:    pcb9612c6ed0c518b.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/mridang/traefik-superheader"] = plugin{
		Create: p910cc03d4a1c785a.CreateConfig,
		New:    p910cc03d4a1c785a.New,
		Version: "v1.1.0",
	}

	pluginMap["github.com/MrNinso/statusdonrouters"] = plugin{
		Create: p209253b91557c632.CreateConfig,
		New:    p209253b91557c632.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/msgbyte/traefik-tianji-plugin"] = plugin{
		Create: pb4e306b64835f3a1.CreateConfig,
		New:    pb4e306b64835f3a1.New,
		Version: "v0.2.1",
	}

	pluginMap["github.com/mubashiroliyantakath/toi"] = plugin{
		Create: pe33efeca394726e0.CreateConfig,
		New:    pe33efeca394726e0.New,
		Version: "v0.2.1",
	}

	pluginMap["github.com/muhgumus/traefik-token-middleware"] = plugin{
		Create: pfa8259cfd86f64a8.CreateConfig,
		New:    pfa8259cfd86f64a8.New,
		Version: "v0.1.13",
	}

	pluginMap["github.com/music-tribe/azadjwtvalidation"] = plugin{
		Create: p939b231a906a460d.CreateConfig,
		New:    p939b231a906a460d.New,
		Version: "v1.1.2",
	}

	pluginMap["github.com/MuXiu1997/traefik-github-oauth-plugin"] = plugin{
		Create: p2b2a64c7ab8851a1.CreateConfig,
		New:    p2b2a64c7ab8851a1.New,
		Version: "v0.3.1",
	}

	pluginMap["github.com/n0m4dz/jwt-cors"] = plugin{
		Create: p69c6463c7b72a85.CreateConfig,
		New:    p69c6463c7b72a85.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/n2jsoft-public-org/traefik-maintenance-plugin"] = plugin{
		Create: pe212d46aa8e78f2a.CreateConfig,
		New:    pe212d46aa8e78f2a.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/ndelta0/interactionverifier"] = plugin{
		Create: p3308929b14214b21.CreateConfig,
		New:    p3308929b14214b21.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/negasus/traefik-plugin-bridge"] = plugin{
		Create: p4a0e6a5eb81c240b.CreateConfig,
		New:    p4a0e6a5eb81c240b.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/negasus/traefik-plugin-ip2location"] = plugin{
		Create: p53e4e22c469731.CreateConfig,
		New:    p53e4e22c469731.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/neggles/middleflare"] = plugin{
		Create: pb03802630afad5b1.CreateConfig,
		New:    pb03802630afad5b1.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/NenoxAG/traefikrealip"] = plugin{
		Create: pf35f61fe67c2b357.CreateConfig,
		New:    pf35f61fe67c2b357.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/nermolaev/traefik-request-id-short"] = plugin{
		Create: pd8aa60efc058c16f.CreateConfig,
		New:    pd8aa60efc058c16f.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/nese/forwardcookie"] = plugin{
		Create: p40d44f72db759626.CreateConfig,
		New:    p40d44f72db759626.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/NETCOREXT/traefik-plugin-response-cache-control"] = plugin{
		Create: p68812a0b5fe6c03b.CreateConfig,
		New:    p68812a0b5fe6c03b.New,
		Version: "v1.0.0-beta.1",
	}

	pluginMap["github.com/Netsocs-Team/keycloakopenid"] = plugin{
		Create: pb4d93615c0bfc3ac.CreateConfig,
		New:    pb4d93615c0bfc3ac.New,
		Version: "v1.0.7",
	}

	pluginMap["github.com/Netsocs-Team/netsocsplugin"] = plugin{
		Create: pafe414e47a8ec48f.CreateConfig,
		New:    pafe414e47a8ec48f.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/Netsocs-Team/traefik-owasp-security"] = plugin{
		Create: pb122cbea7a1b3e5e.CreateConfig,
		New:    pb122cbea7a1b3e5e.New,
		Version: "v1.3.8",
	}

	pluginMap["github.com/Netvigie/traefik-json-body-validator"] = plugin{
		Create: pba20f0c4e69d.CreateConfig,
		New:    pba20f0c4e69d.New,
		Version: "v1.0.5",
	}

	pluginMap["github.com/neuraflow-github/my-traefik-jwt-plugin"] = plugin{
		Create: p95b73f28e84d19a8.CreateConfig,
		New:    p95b73f28e84d19a8.New,
		Version: "v1.0.4",
	}

	pluginMap["github.com/ngocdv86/plugin-rewritebody"] = plugin{
		Create: p84e256f618109450.CreateConfig,
		New:    p84e256f618109450.New,
		Version: "v1.0.16",
	}

	pluginMap["github.com/ngocdv86/rate-limit"] = plugin{
		Create: pf74a39beb0e04de5.CreateConfig,
		New:    pf74a39beb0e04de5.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/nhomchatgpt/headerblock"] = plugin{
		Create: p4f0537174face82f.CreateConfig,
		New:    p4f0537174face82f.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/NiklasPor/traefik-plugin-replace-query-regex"] = plugin{
		Create: p6e7610081c50357f.CreateConfig,
		New:    p6e7610081c50357f.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/nilskohrs/environmentheader"] = plugin{
		Create: p99adb2b0247f21cd.CreateConfig,
		New:    p99adb2b0247f21cd.New,
		Version: "v0.1.4",
	}

	pluginMap["github.com/nilskohrs/headerblock"] = plugin{
		Create: pf3908b3757db92af.CreateConfig,
		New:    pf3908b3757db92af.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/nilskohrs/pathauth"] = plugin{
		Create: pd65bb09b2e7e629d.CreateConfig,
		New:    pd65bb09b2e7e629d.New,
		Version: "v0.1.5",
	}

	pluginMap["github.com/nilskohrs/regex2redirect"] = plugin{
		Create: p706e78ba34ab4fd6.CreateConfig,
		New:    p706e78ba34ab4fd6.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/nilskohrs/reproxied"] = plugin{
		Create: pd13b8fbb49d46f57.CreateConfig,
		New:    pd13b8fbb49d46f57.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/nilskohrs/stripcookie"] = plugin{
		Create: p3a31d7994625b7a5.CreateConfig,
		New:    p3a31d7994625b7a5.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/Noahnut/replacePathRegex"] = plugin{
		Create: pae6ea5e7e867358a.CreateConfig,
		New:    pae6ea5e7e867358a.New,
		Version: "v0.1.5",
	}

	pluginMap["github.com/noaHson86/signature-plugin"] = plugin{
		Create: pcdae6056c5880760.CreateConfig,
		New:    pcdae6056c5880760.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/NovinSystemCom/identityplugin"] = plugin{
		Create: peff5071eeef35ab5.CreateConfig,
		New:    peff5071eeef35ab5.New,
		Version: "v0.1.4",
	}

	pluginMap["github.com/nscuro/traefik-plugin-geoblock"] = plugin{
		Create: p1928035367dedded.CreateConfig,
		New:    p1928035367dedded.New,
		Version: "v0.14.0",
	}

	pluginMap["github.com/NX211/traefik-proxmox-provider"] = plugin{
		Create: paf1271f54f4fbf82.CreateConfig,
		New:    paf1271f54f4fbf82.New,
		Version: "v0.7.6",
	}

	pluginMap["github.com/NX211/traefik-webfinger"] = plugin{
		Create: p21f01d20e90a15dd.CreateConfig,
		New:    p21f01d20e90a15dd.New,
		Version: "v0.3.5",
	}

	pluginMap["github.com/nzin/traefik-cluster-ratelimit"] = plugin{
		Create: p5a3a3b40224f98d2.CreateConfig,
		New:    p5a3a3b40224f98d2.New,
		Version: "v1.1.1",
	}

	pluginMap["github.com/omar-shrbajy-arive/headerauthentication"] = plugin{
		Create: p5825f722c9865ad.CreateConfig,
		New:    p5825f722c9865ad.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/opaas-cloud/traefik-plugin-proxy-cookie"] = plugin{
		Create: pe8b769d9ad8ec2fc.CreateConfig,
		New:    pe8b769d9ad8ec2fc.New,
		Version: "v1.4.2",
	}

	pluginMap["github.com/openware/barongz"] = plugin{
		Create: p9150e36ea73bcaf4.CreateConfig,
		New:    p9150e36ea73bcaf4.New,
		Version: "0.0.1",
	}

	pluginMap["github.com/packruler/rewrite-body"] = plugin{
		Create: p96a63e9a9286a05a.CreateConfig,
		New:    p96a63e9a9286a05a.New,
		Version: "v1.2.0",
	}

	pluginMap["github.com/packruler/traefik-themepark"] = plugin{
		Create: p33d04b4cd62e4007.CreateConfig,
		New:    p33d04b4cd62e4007.New,
		Version: "v1.4.2",
	}

	pluginMap["github.com/paladium/traefikkeycloak"] = plugin{
		Create: pfa6ad40c666df602.CreateConfig,
		New:    pfa6ad40c666df602.New,
		Version: "v1.11",
	}

	pluginMap["github.com/pamdigitek-doomo/traefik-jwt"] = plugin{
		Create: paa1cf5af57ddab7.CreateConfig,
		New:    paa1cf5af57ddab7.New,
		Version: "v0.1.16",
	}

	pluginMap["github.com/PandaWorker/traefik-upstream-when"] = plugin{
		Create: pdd854c089543d1f5.CreateConfig,
		New:    pdd854c089543d1f5.New,
		Version: "v0.1.4",
	}

	pluginMap["github.com/Papercast-Limited/epoch"] = plugin{
		Create: p6d20c1589d2f550f.CreateConfig,
		New:    p6d20c1589d2f550f.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/PascalMinder/geoblock"] = plugin{
		Create: p942d236640fca508.CreateConfig,
		New:    p942d236640fca508.New,
		Version: "v0.3.3",
	}

	pluginMap["github.com/PatrickMi/body-forward-auth"] = plugin{
		Create: pabc7bba81e2d3e9f.CreateConfig,
		New:    pabc7bba81e2d3e9f.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/paul-vautier/enieca"] = plugin{
		Create: p3e36be5d0844066b.CreateConfig,
		New:    p3e36be5d0844066b.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/PaulLeRoux142/TorBlockRedirect"] = plugin{
		Create: p37d4a04837503f7b.CreateConfig,
		New:    p37d4a04837503f7b.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/pavankumar0143/traefik-lambdaauthorizer"] = plugin{
		Create: p2b4720f85cb5b4ec.CreateConfig,
		New:    p2b4720f85cb5b4ec.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/pavankumar0143/traefik-lambdarequesttransformer"] = plugin{
		Create: p2c87f3bb7183eaf3.CreateConfig,
		New:    p2c87f3bb7183eaf3.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/pavankumar0143/traefik-lambdaresponsetransformer"] = plugin{
		Create: p1f774d5dba6db4de.CreateConfig,
		New:    p1f774d5dba6db4de.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/PavloZastavnyi/headerstransformation"] = plugin{
		Create: p58f7121716c1b4dd.CreateConfig,
		New:    p58f7121716c1b4dd.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/Paxxs/traefik-get-real-ip"] = plugin{
		Create: p7ad222b5e1b4b78b.CreateConfig,
		New:    p7ad222b5e1b4b78b.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/pdazcom/botdetector"] = plugin{
		Create: pc59e4212810de2d3.CreateConfig,
		New:    pc59e4212810de2d3.New,
		Version: "v0.4.1",
	}

	pluginMap["github.com/Penitence1992/traefik-ldap-plugin"] = plugin{
		Create: p4fb28aefa785a324.CreateConfig,
		New:    p4fb28aefa785a324.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/pierre-verhaeghe/traefik-replace-response-code"] = plugin{
		Create: pb88cf4f810b06767.CreateConfig,
		New:    pb88cf4f810b06767.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/PingThingsIO/traefik-jwt-group-access"] = plugin{
		Create: pf84d65a6e29af10b.CreateConfig,
		New:    pf84d65a6e29af10b.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/pipe01/plugin-requestid"] = plugin{
		Create: pb67997b599b14d41.CreateConfig,
		New:    pb67997b599b14d41.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/Pival81/keycloakopenid"] = plugin{
		Create: p18c09b501bfc27d5.CreateConfig,
		New:    p18c09b501bfc27d5.New,
		Version: "v0.1.37",
	}

	pluginMap["github.com/pnxs/traefik-plugin-mtls-header"] = plugin{
		Create: pbe330377648f5a65.CreateConfig,
		New:    pbe330377648f5a65.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/poloyacero/headauth"] = plugin{
		Create: p2a4d577c55d0442e.CreateConfig,
		New:    p2a4d577c55d0442e.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/PongDev/traefikbodytransform"] = plugin{
		Create: p9981954b6bd1872f.CreateConfig,
		New:    p9981954b6bd1872f.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/portbrella/traefik_whitelist"] = plugin{
		Create: pcb9779f11f59a70b.CreateConfig,
		New:    pcb9779f11f59a70b.New,
		Version: "v1.0.6",
	}

	pluginMap["github.com/portofrotterdam/environmentheader"] = plugin{
		Create: pb617fb8fcf2611e0.CreateConfig,
		New:    pb617fb8fcf2611e0.New,
		Version: "v0.1.4",
	}

	pluginMap["github.com/portofrotterdam/environmentpathappender"] = plugin{
		Create: p5bcf4c2db72b424c.CreateConfig,
		New:    p5bcf4c2db72b424c.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/portofrotterdam/headerblock"] = plugin{
		Create: p564ff97e39e1ce8a.CreateConfig,
		New:    p564ff97e39e1ce8a.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/portofrotterdam/pathauth"] = plugin{
		Create: p5b63e3d053a0302.CreateConfig,
		New:    p5b63e3d053a0302.New,
		Version: "v0.1.5",
	}

	pluginMap["github.com/portofrotterdam/regex2redirect"] = plugin{
		Create: p8ea3a6b4720064a5.CreateConfig,
		New:    p8ea3a6b4720064a5.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/portofrotterdam/reproxied"] = plugin{
		Create: pee0a0fef92696f77.CreateConfig,
		New:    pee0a0fef92696f77.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/portofrotterdam/stripcookie"] = plugin{
		Create: p5b116bd5f754ca84.CreateConfig,
		New:    p5b116bd5f754ca84.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/portswigger-cloud/cloudfrontgate"] = plugin{
		Create: p3a31907a2dc7fb23.CreateConfig,
		New:    p3a31907a2dc7fb23.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/portswigger-cloud/requestsenderplugin"] = plugin{
		Create: pc1bf5d27f93762c9.CreateConfig,
		New:    pc1bf5d27f93762c9.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/PRIHLOP/traefik-body-rewrite"] = plugin{
		Create: p84e82d5c03696d04.CreateConfig,
		New:    p84e82d5c03696d04.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/programic/traefik-maintenance-plugin"] = plugin{
		Create: pefd75e9bd580a75a.CreateConfig,
		New:    pefd75e9bd580a75a.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/project-echo/traefik-ocsp"] = plugin{
		Create: p9fa1167301e95ba6.CreateConfig,
		New:    p9fa1167301e95ba6.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/przemek-carma/w3c-traceparent-generator"] = plugin{
		Create: p1280d9eefca9c249.CreateConfig,
		New:    p1280d9eefca9c249.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/PseudoResonance/cloudflarewarp"] = plugin{
		Create: p73dc56df3ffe6733.CreateConfig,
		New:    p73dc56df3ffe6733.New,
		Version: "v1.4.2",
	}

	pluginMap["github.com/PseudoResonance/traefikerrorreplace"] = plugin{
		Create: pa9b721c8b3c93527.CreateConfig,
		New:    pa9b721c8b3c93527.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/psncius/traefik-api-middleware"] = plugin{
		Create: p7d845c03e236993.CreateConfig,
		New:    p7d845c03e236993.New,
		Version: "v0.1.4",
	}

	pluginMap["github.com/pvalletbo/traefik-blocklist"] = plugin{
		Create: pc88f3cb6abdf1d8b.CreateConfig,
		New:    pc88f3cb6abdf1d8b.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/pvalletbo/traefik-forwarded-real-ip"] = plugin{
		Create: pbd9895c860d1dde0.CreateConfig,
		New:    pbd9895c860d1dde0.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/pvliesdonk/mtlsforward"] = plugin{
		Create: p2fdf76ce2145c822.CreateConfig,
		New:    p2fdf76ce2145c822.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/pxxonline/traefik-plugin-cors"] = plugin{
		Create: pbe1731e22087a7a0.CreateConfig,
		New:    pbe1731e22087a7a0.New,
		Version: "v0.1.7",
	}

	pluginMap["github.com/pyksid/cloudflarewarp"] = plugin{
		Create: pa0c8fc75db8b2c08.CreateConfig,
		New:    pa0c8fc75db8b2c08.New,
		Version: "v1.3.4",
	}

	pluginMap["github.com/pyrho/badgerheaders"] = plugin{
		Create: p5d98499a72392805.CreateConfig,
		New:    p5d98499a72392805.New,
		Version: "v0.0.12",
	}

	pluginMap["github.com/quintinheard/traefik-cors/traefik"] = plugin{
		Create: pb788e338709619ef.CreateConfig,
		New:    pb788e338709619ef.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/quortex/traefik-responsebodyrewrite"] = plugin{
		Create: pe36d636ac3559a7e.CreateConfig,
		New:    pe36d636ac3559a7e.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/quortex/traefik-responseheadersfilter"] = plugin{
		Create: pd999b002eb65af1.CreateConfig,
		New:    pd999b002eb65af1.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/qwercik/traefik-original-uri"] = plugin{
		Create: p8fcb624c8690d057.CreateConfig,
		New:    p8fcb624c8690d057.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/qxsugar/request-dispatch"] = plugin{
		Create: pcf17fd66255f650a.CreateConfig,
		New:    pcf17fd66255f650a.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/qxsugar/request-mark"] = plugin{
		Create: p819d703205fd6192.CreateConfig,
		New:    p819d703205fd6192.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/qxsugar/traefik-jwt-parser"] = plugin{
		Create: pf98836f5f6059090.CreateConfig,
		New:    pf98836f5f6059090.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/r3nic1e/traefik-plugin-add-response-header"] = plugin{
		Create: pa5483386e15bf56a.CreateConfig,
		New:    pa5483386e15bf56a.New,
		Version: "v0.5.1",
	}

	pluginMap["github.com/rafal-slowik/traceparent-plugin"] = plugin{
		Create: p42c3b65c6b878eb2.CreateConfig,
		New:    p42c3b65c6b878eb2.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/Rajabalian/ipclient"] = plugin{
		Create: p503af66018e4063e.CreateConfig,
		New:    p503af66018e4063e.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/Rau-N/DomainSentinel"] = plugin{
		Create: pb341ca3bd0670116.CreateConfig,
		New:    pb341ca3bd0670116.New,
		Version: "v1.1.0",
	}

	pluginMap["github.com/renanqts/xdpfail2ban"] = plugin{
		Create: pae6a54e4192c57ff.CreateConfig,
		New:    pae6a54e4192c57ff.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/rhabichl/applicationgatewaywhitelist"] = plugin{
		Create: pc9caaca396dda32.CreateConfig,
		New:    pc9caaca396dda32.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/Ridecell/traefik-token-checker"] = plugin{
		Create: pcf490a990232afff.CreateConfig,
		New:    pcf490a990232afff.New,
		Version: "v0.0.11",
	}

	pluginMap["github.com/rinokadijk/traefik-api-key"] = plugin{
		Create: p5e339e5c331a7c2b.CreateConfig,
		New:    p5e339e5c331a7c2b.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/rinokadijk/traefik-openai-header"] = plugin{
		Create: p3bb88aaab1ae0b55.CreateConfig,
		New:    p3bb88aaab1ae0b55.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/RiskIdent/traefik-remoteaddr-plugin"] = plugin{
		Create: p4e0c1092a500dada.CreateConfig,
		New:    p4e0c1092a500dada.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/RiskIdent/traefik-tls-headers-plugin"] = plugin{
		Create: p5d88a78ef087cca7.CreateConfig,
		New:    p5d88a78ef087cca7.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/rjop-hccgt/traefik-forward-slash-redirector"] = plugin{
		Create: p8e0fd2112689537b.CreateConfig,
		New:    p8e0fd2112689537b.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/rjop-hccgt/traefikpluginhcindex"] = plugin{
		Create: pe5d3bb9d60f17b58.CreateConfig,
		New:    pe5d3bb9d60f17b58.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/rocdove/replacepathfromurlregex"] = plugin{
		Create: p7fc777d3465e3f9.CreateConfig,
		New:    p7fc777d3465e3f9.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/romracer/traefik-get-real-ip"] = plugin{
		Create: p65f68000846c473b.CreateConfig,
		New:    p65f68000846c473b.New,
		Version: "v1.0.2-1",
	}

	pluginMap["github.com/RouxAntoine/reproxied"] = plugin{
		Create: pb517e0631b589fba.CreateConfig,
		New:    pb517e0631b589fba.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/RSS3-Network/gatewayflowcontroller"] = plugin{
		Create: p958c256be0fa43d3.CreateConfig,
		New:    p958c256be0fa43d3.New,
		Version: "v0.0.6",
	}

	pluginMap["github.com/russ-p/traefik-plugin-static-sites"] = plugin{
		Create: p2cf9bbfd1fd2f241.CreateConfig,
		New:    p2cf9bbfd1fd2f241.New,
		Version: "v0.1.5",
	}

	pluginMap["github.com/Russia9/body-size-limit"] = plugin{
		Create: pf78f3c90684ef7fb.CreateConfig,
		New:    pf78f3c90684ef7fb.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/sablierapp/sablier/plugins/traefik"] = plugin{
		Create: pa41dd68ff53e1b51.CreateConfig,
		New:    pa41dd68ff53e1b51.New,
		Version: "v1.10.1",
	}

	pluginMap["github.com/sadaghiani/traefik-auth-middleware"] = plugin{
		Create: p7457e5b728f06a3d.CreateConfig,
		New:    p7457e5b728f06a3d.New,
		Version: "v1.1.1",
	}

	pluginMap["github.com/safing/plausiblefeeder"] = plugin{
		Create: p845fb125f88f36d4.CreateConfig,
		New:    p845fb125f88f36d4.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/safing/scanblock"] = plugin{
		Create: pfb0acf2c38e4b068.CreateConfig,
		New:    pfb0acf2c38e4b068.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/safing/tlsauth"] = plugin{
		Create: p1cd37128322c621c.CreateConfig,
		New:    p1cd37128322c621c.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/sagarrakshe/b64-header-parser"] = plugin{
		Create: p8e27bda7b3ebdbc4.CreateConfig,
		New:    p8e27bda7b3ebdbc4.New,
		Version: "v1.0.4",
	}

	pluginMap["github.com/saltyorg/cloudflarewarp"] = plugin{
		Create: p51da206482a8d06a.CreateConfig,
		New:    p51da206482a8d06a.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/saman-jafari/correlation-id-traefik"] = plugin{
		Create: p737fa363d94e1024.CreateConfig,
		New:    p737fa363d94e1024.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/samerbahri98/sigv4middleware"] = plugin{
		Create: pfbe4cc0d107f41f3.CreateConfig,
		New:    pfbe4cc0d107f41f3.New,
		Version: "v0.1.7",
	}

	pluginMap["github.com/sanderPostma/traefik-validate-jwt"] = plugin{
		Create: p453b8b3875b99c8.CreateConfig,
		New:    p453b8b3875b99c8.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/sasd13/traefik-keycloak-authorizer"] = plugin{
		Create: pbdd00c6440ab73a1.CreateConfig,
		New:    pbdd00c6440ab73a1.New,
		Version: "v0.2.1",
	}

	pluginMap["github.com/sasd13/traefik-proxy-forward"] = plugin{
		Create: p438ae091c0f46beb.CreateConfig,
		New:    p438ae091c0f46beb.New,
		Version: "v0.3.1",
	}

	pluginMap["github.com/sasd13/traefik-proxy-header"] = plugin{
		Create: p1c7269fb4651c8fe.CreateConfig,
		New:    p1c7269fb4651c8fe.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/schackoa/replacepathfromurlregex"] = plugin{
		Create: pc6f76e54ef6731ad.CreateConfig,
		New:    pc6f76e54ef6731ad.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/SchmitzDan/traefik-plugin-cookie-path-prefix"] = plugin{
		Create: p39807db6be088b87.CreateConfig,
		New:    p39807db6be088b87.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/SchmitzDan/traefik-plugin-proxy-cookie"] = plugin{
		Create: p77f48e1c385a89ba.CreateConfig,
		New:    p77f48e1c385a89ba.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/SchmitzDan/traefik-plugin-redirect-location"] = plugin{
		Create: pc14b69c1ab6de0a1.CreateConfig,
		New:    pc14b69c1ab6de0a1.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/scrazy77/customerrorsrewrite"] = plugin{
		Create: pf347bc57ae317a3d.CreateConfig,
		New:    pf347bc57ae317a3d.New,
		Version: "v0.0.8",
	}

	pluginMap["github.com/scrazy77/dragonfly2imgproxy"] = plugin{
		Create: p95771a3e7bf7078c.CreateConfig,
		New:    p95771a3e7bf7078c.New,
		Version: "v0.0.22",
	}

	pluginMap["github.com/scrazy77/plugin-simplecache-nocache"] = plugin{
		Create: pb6ae1ae2ce01044a.CreateConfig,
		New:    pb6ae1ae2ce01044a.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/Sensedia/traefik-plugin-decompress"] = plugin{
		Create: p1d971293c9758a13.CreateConfig,
		New:    p1d971293c9758a13.New,
		Version: "v1.0.5",
	}

	pluginMap["github.com/Septima/traefik-api-key-auth"] = plugin{
		Create: p4e2178d04d6c0c2d.CreateConfig,
		New:    p4e2178d04d6c0c2d.New,
		Version: "v0.3.0",
	}

	pluginMap["github.com/SergioFloresG/corsmiddleware"] = plugin{
		Create: pa6f30c66d795bf15.CreateConfig,
		New:    pa6f30c66d795bf15.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/set-de/jwt-middleware"] = plugin{
		Create: paf635593a48897fe.CreateConfig,
		New:    paf635593a48897fe.New,
		Version: "v1.2.6",
	}

	pluginMap["github.com/sevensolutions/traefik-oidc-auth/src"] = plugin{
		Create: p75afbeb31ee3f53.CreateConfig,
		New:    p75afbeb31ee3f53.New,
		Version: "v0.16.0",
	}

	pluginMap["github.com/sevensolutions/traefik-plugin-structure-demo/src"] = plugin{
		Create: pe9ace569cefce718.CreateConfig,
		New:    pe9ace569cefce718.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/shantanugadgil/traefik-block-regex-urls"] = plugin{
		Create: pd3eb13e8699c1e40.CreateConfig,
		New:    pd3eb13e8699c1e40.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/ShaunVyxw/my_plugin"] = plugin{
		Create: p82a2221a57835821.CreateConfig,
		New:    p82a2221a57835821.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/Shoggomo/traefik_dynamic_public_whitelist"] = plugin{
		Create: pd51dbe5489cb4790.CreateConfig,
		New:    pd51dbe5489cb4790.New,
		Version: "v0.1.5",
	}

	pluginMap["github.com/SimpaiX-net/traefik-guard"] = plugin{
		Create: p95bea78df0849304.CreateConfig,
		New:    p95bea78df0849304.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/skynet2/traefik-fallback-plugin"] = plugin{
		Create: p6b46b694e2448dc.CreateConfig,
		New:    p6b46b694e2448dc.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/slimani-dev/dynamichost"] = plugin{
		Create: pdf6a64b0c011d7d1.CreateConfig,
		New:    pdf6a64b0c011d7d1.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/smerschjohann/mtlswhitelist"] = plugin{
		Create: p64b24807a23098d4.CreateConfig,
		New:    p64b24807a23098d4.New,
		Version: "v0.3.0",
	}

	pluginMap["github.com/snapt/traefik-nova-plugin"] = plugin{
		Create: pdf6a887bfe40cfa2.CreateConfig,
		New:    pdf6a887bfe40cfa2.New,
		Version: "v1.1.2",
	}

	pluginMap["github.com/softwaremastermind/defaultcspheader"] = plugin{
		Create: p1d999d72b46bdf7b.CreateConfig,
		New:    p1d999d72b46bdf7b.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/solution-libre/traefik-plugin-robots-txt"] = plugin{
		Create: pebe2f99c694e50db.CreateConfig,
		New:    pebe2f99c694e50db.New,
		Version: "v0.2.1",
	}

	pluginMap["github.com/soulbalz/correlationid"] = plugin{
		Create: p1075e2fe49e6934d.CreateConfig,
		New:    p1075e2fe49e6934d.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/soulbalz/traefik-check-body"] = plugin{
		Create: p4835087f73fb154b.CreateConfig,
		New:    p4835087f73fb154b.New,
		Version: "v1.1.2",
	}

	pluginMap["github.com/soulbalz/traefik-real-ip"] = plugin{
		Create: p822ad9b7eb8fea8e.CreateConfig,
		New:    p822ad9b7eb8fea8e.New,
		Version: "v1.0.3",
	}

	pluginMap["github.com/sp-jcberleur/xrequesttrace"] = plugin{
		Create: p59f320025e1cfdc6.CreateConfig,
		New:    p59f320025e1cfdc6.New,
		Version: "v0.1.6",
	}

	pluginMap["github.com/Spakl-io/shorty"] = plugin{
		Create: p534c77c64403e1e.CreateConfig,
		New:    p534c77c64403e1e.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/sproutmaster/TraefikIPRules"] = plugin{
		Create: p63149cb160890c2c.CreateConfig,
		New:    p63149cb160890c2c.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/sstoner/cloudflaregate"] = plugin{
		Create: p521a168be3d470fd.CreateConfig,
		New:    p521a168be3d470fd.New,
		Version: "v1.1.2",
	}

	pluginMap["github.com/stabelo/traefik-tracking-cookie"] = plugin{
		Create: pc9de36c5de123c83.CreateConfig,
		New:    pc9de36c5de123c83.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/steveiliop56/tinyrobotsblock"] = plugin{
		Create: p4b593cdbb54d6846.CreateConfig,
		New:    p4b593cdbb54d6846.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/strigo/traefik-auth-middleware"] = plugin{
		Create: p134eb94002c89824.CreateConfig,
		New:    p134eb94002c89824.New,
		Version: "v0.2.1",
	}

	pluginMap["github.com/subotaii/traefik-plugin-addprefix-from-host"] = plugin{
		Create: p9f6e10ec13ca7ab4.CreateConfig,
		New:    p9f6e10ec13ca7ab4.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/sunalwaysknows/redirect2https"] = plugin{
		Create: p4c466ab25601f7b4.CreateConfig,
		New:    p4c466ab25601f7b4.New,
		Version: "v0.0.7",
	}

	pluginMap["github.com/supergoudvis116/regex-redirect-joule"] = plugin{
		Create: p1f0bd9f0f33221.CreateConfig,
		New:    p1f0bd9f0f33221.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/suteqa/plugin_record"] = plugin{
		Create: p98c5c6a64d8d5c6d.CreateConfig,
		New:    p98c5c6a64d8d5c6d.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/sw360cab/cncftaeplugin"] = plugin{
		Create: p3d0a396678cc4ba2.CreateConfig,
		New:    p3d0a396678cc4ba2.New,
		Version: "0.0.3",
	}

	pluginMap["github.com/SwissDataScienceCenter/cookiefilter"] = plugin{
		Create: p7d7cab2fea196357.CreateConfig,
		New:    p7d7cab2fea196357.New,
		Version: "0.0.2",
	}

	pluginMap["github.com/sysradium/traefik-request-signature-verifier"] = plugin{
		Create: p55267e921164c7c2.CreateConfig,
		New:    p55267e921164c7c2.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/taskmedia/ddns-allowlist"] = plugin{
		Create: pbbe71717d493b297.CreateConfig,
		New:    pbbe71717d493b297.New,
		Version: "v1.7.0",
	}

	pluginMap["github.com/taskmedia/ddns-whitelist"] = plugin{
		Create: p9632062027b1f09b.CreateConfig,
		New:    p9632062027b1f09b.New,
		Version: "v1.3.0",
	}

	pluginMap["github.com/tdilber/anouncy-traefik-plugin"] = plugin{
		Create: p2d51f97b32991267.CreateConfig,
		New:    p2d51f97b32991267.New,
		Version: "v0.0.6",
	}

	pluginMap["github.com/TDL-Bewatec/traefikbodytransform"] = plugin{
		Create: pe857d533a3c7416c.CreateConfig,
		New:    pe857d533a3c7416c.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/team-carepay/traefik-jwt-plugin"] = plugin{
		Create: pa1b3b93e672dfb70.CreateConfig,
		New:    pa1b3b93e672dfb70.New,
		Version: "v0.6.0",
	}

	pluginMap["github.com/team-carepay/traefik-opa-plugin"] = plugin{
		Create: p3bbebc73b1696bac.CreateConfig,
		New:    p3bbebc73b1696bac.New,
		Version: "v0.0.3",
	}

	pluginMap["github.com/TechAlchemistry/traefik-maintenance-warden"] = plugin{
		Create: p7899ba08037c8df7.CreateConfig,
		New:    p7899ba08037c8df7.New,
		Version: "v1.1.1",
	}

	pluginMap["github.com/tgrosinger/obsidian-publish-traefik-middleware"] = plugin{
		Create: p8e654159c8c4d52c.CreateConfig,
		New:    p8e654159c8c4d52c.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/the-ccsn/traefik-plugin-rewritebody"] = plugin{
		Create: p2c351dbf32dfa7a5.CreateConfig,
		New:    p2c351dbf32dfa7a5.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/theoguidoux/cookiesmanager"] = plugin{
		Create: pa7337476a94a3cd1.CreateConfig,
		New:    pa7337476a94a3cd1.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/thiagotognoli/traefikgeoip"] = plugin{
		Create: p21804cd5b3ce5e7a.CreateConfig,
		New:    p21804cd5b3ce5e7a.New,
		Version: "v1.1.1",
	}

	pluginMap["github.com/Thijmen/traefik-query-parameters-middleware"] = plugin{
		Create: p6ac8c5a73011f480.CreateConfig,
		New:    p6ac8c5a73011f480.New,
		Version: "v1.1.1",
	}

	pluginMap["github.com/Thijmen/traefik-remove-query-parameters-by-regex"] = plugin{
		Create: p72063f9727be73cd.CreateConfig,
		New:    p72063f9727be73cd.New,
		Version: "v1.1.0",
	}

	pluginMap["github.com/TicketGenieIO/plugin_forwardedauth"] = plugin{
		Create: pc87c2241fa1ffbf2.CreateConfig,
		New:    pc87c2241fa1ffbf2.New,
		Version: "v1.0.6",
	}

	pluginMap["github.com/tilak999/traefikplugin"] = plugin{
		Create: p8678aebe2bb033c2.CreateConfig,
		New:    p8678aebe2bb033c2.New,
		Version: "v0.0.8",
	}

	pluginMap["github.com/tkreiner/traefik-regex-block"] = plugin{
		Create: p15f9da7fb302e52c.CreateConfig,
		New:    p15f9da7fb302e52c.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/tmpim/tmpauth-traefik"] = plugin{
		Create: p277091e08872b7af.CreateConfig,
		New:    p277091e08872b7af.New,
		Version: "v0.3",
	}

	pluginMap["github.com/tnt-sbab/jwt-verifier"] = plugin{
		Create: pbe5cdb2421ed3add.CreateConfig,
		New:    pbe5cdb2421ed3add.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/tnt-sbab/token-translator"] = plugin{
		Create: peadef30a10c3ee95.CreateConfig,
		New:    peadef30a10c3ee95.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/toanz/jwt-token"] = plugin{
		Create: p53e83b240d802b7d.CreateConfig,
		New:    p53e83b240d802b7d.New,
		Version: "v0.1.4",
	}

	pluginMap["github.com/toanz/traefik-plugin-add-response-header"] = plugin{
		Create: pb6d67c091f2e1a2d.CreateConfig,
		New:    pb6d67c091f2e1a2d.New,
		Version: "v0.5.1",
	}

	pluginMap["github.com/togettoyou/traefik-timer-plugin"] = plugin{
		Create: p600e93237b61d443.CreateConfig,
		New:    p600e93237b61d443.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/tommoulard/fail2ban"] = plugin{
		Create: p8001fc243a0e7be.CreateConfig,
		New:    p8001fc243a0e7be.New,
		Version: "v0.6.2",
	}

	pluginMap["github.com/tomMoulard/traefik-plugin-waeb"] = plugin{
		Create: pc93e97832d3e6291.CreateConfig,
		New:    pc93e97832d3e6291.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/tonyfud/traefikjwttoken"] = plugin{
		Create: p71133c721994b704.CreateConfig,
		New:    p71133c721994b704.New,
		Version: "v0.0.6",
	}

	pluginMap["github.com/tpaulus/jwt-middleware"] = plugin{
		Create: p2d0e7c7c3c6b290f.CreateConfig,
		New:    p2d0e7c7c3c6b290f.New,
		Version: "v1.1.13",
	}

	pluginMap["github.com/Traceableai/traceableai_traefik_plugin"] = plugin{
		Create: pbc3d3536ae746bc6.CreateConfig,
		New:    pbc3d3536ae746bc6.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/traefik-contrib/noop"] = plugin{
		Create: pe259452c5816c5c8.CreateConfig,
		New:    pe259452c5816c5c8.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/traefik-plugins/traefik-jwt-plugin"] = plugin{
		Create: pe6c37f1e7f02a54f.CreateConfig,
		New:    pe6c37f1e7f02a54f.New,
		Version: "v0.10.0",
	}

	pluginMap["github.com/traefik-plugins/traefikgeoip2"] = plugin{
		Create: p7a60b7ee5412953a.CreateConfig,
		New:    p7a60b7ee5412953a.New,
		Version: "v0.22.0",
	}

	pluginMap["github.com/traefik-plugins/traefikuseragent"] = plugin{
		Create: p1695a899b982abc5.CreateConfig,
		New:    p1695a899b982abc5.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/traefik/plugin-blockpath"] = plugin{
		Create: p68117edcfbcf4706.CreateConfig,
		New:    p68117edcfbcf4706.New,
		Version: "v0.2.1",
	}

	pluginMap["github.com/traefik/plugin-log4shell"] = plugin{
		Create: p619dce1d4f0b1c30.CreateConfig,
		New:    p619dce1d4f0b1c30.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/traefik/plugin-rewritebody"] = plugin{
		Create: pecd51dbd1979b8c5.CreateConfig,
		New:    pecd51dbd1979b8c5.New,
		Version: "v0.3.1",
	}

	pluginMap["github.com/traefik/plugin-simplecache"] = plugin{
		Create: p2ad1e6a65fe3d90c.CreateConfig,
		New:    p2ad1e6a65fe3d90c.New,
		Version: "v0.2.1",
	}

	pluginMap["github.com/traefik/plugindemo"] = plugin{
		Create: p10de222177ac8e3d.CreateConfig,
		New:    p10de222177ac8e3d.New,
		Version: "v0.2.2",
	}

	pluginMap["github.com/traefik/pluginproviderdemo"] = plugin{
		Create: pf88cb09c7b357f87.CreateConfig,
		New:    pf88cb09c7b357f87.New,
		Version: "v0.3.0",
	}

	pluginMap["github.com/Treblle/TreblleTraefikPluginGo"] = plugin{
		Create: pdb96acd5de7db9c4.CreateConfig,
		New:    pdb96acd5de7db9c4.New,
		Version: "v1.0.5",
	}

	pluginMap["github.com/TreyWW/traefik-plugin-original-host-header"] = plugin{
		Create: p3bfef9eb9e515126.CreateConfig,
		New:    p3bfef9eb9e515126.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/TRIMM/redirects-traefik-middleware"] = plugin{
		Create: pc4656518e1fa29b1.CreateConfig,
		New:    pc4656518e1fa29b1.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/TRIMM/traefik-maintenance"] = plugin{
		Create: pbc2d85eda113c436.CreateConfig,
		New:    pbc2d85eda113c436.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/trinnylondon/lowercase"] = plugin{
		Create: pad3f4dc55fb58938.CreateConfig,
		New:    pad3f4dc55fb58938.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/trinnylondon/traefik-add-trace-id"] = plugin{
		Create: p481b01b6785e6733.CreateConfig,
		New:    p481b01b6785e6733.New,
		Version: "v0.1.5",
	}

	pluginMap["github.com/trois-six/plugin-httplog"] = plugin{
		Create: pe04bc247cd53be62.CreateConfig,
		New:    pe04bc247cd53be62.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/trois-six/plugin-securelink"] = plugin{
		Create: paf5cb2d07c9d73e2.CreateConfig,
		New:    paf5cb2d07c9d73e2.New,
		Version: "v0.1.7",
	}

	pluginMap["github.com/trolleksii/traefik-plugin-mutate-headers"] = plugin{
		Create: pa163bd1824ec3b6d.CreateConfig,
		New:    pa163bd1824ec3b6d.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/trondhindenes/traefikreplay"] = plugin{
		Create: p1a9a222a5219607e.CreateConfig,
		New:    p1a9a222a5219607e.New,
		Version: "v1.0.8",
	}

	pluginMap["github.com/tuxgal/traefik_inline_response"] = plugin{
		Create: p8b695bef12267275.CreateConfig,
		New:    p8b695bef12267275.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/unbasical/traefik-json-body2header"] = plugin{
		Create: p69de2b3c97d8c975.CreateConfig,
		New:    p69de2b3c97d8c975.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/unnoo/forward-port"] = plugin{
		Create: pb5ed6cb55532db95.CreateConfig,
		New:    pb5ed6cb55532db95.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/unsoon/traefik-open-policy-agent"] = plugin{
		Create: p8bd1b4be9ce24321.CreateConfig,
		New:    p8bd1b4be9ce24321.New,
		Version: "v1.2.1",
	}

	pluginMap["github.com/unsoon/traefik-require-auth-headers"] = plugin{
		Create: p58fb11807ab8c4f4.CreateConfig,
		New:    p58fb11807ab8c4f4.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/usalko/swagger-merge-docs"] = plugin{
		Create: p1a0cafb0f33a0aba.CreateConfig,
		New:    p1a0cafb0f33a0aba.New,
		Version: "v0.1.6",
	}

	pluginMap["github.com/usalko/swagger-ring"] = plugin{
		Create: p1be319526fc5b221.CreateConfig,
		New:    p1be319526fc5b221.New,
		Version: "v0.1.10",
	}

	pluginMap["github.com/Uscreen-video/traefik-plugin-rewritehost"] = plugin{
		Create: p7f2b622d496639e4.CreateConfig,
		New:    p7f2b622d496639e4.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/usegiam/giam-traefik-plugin"] = plugin{
		Create: p20256bde45d79155.CreateConfig,
		New:    p20256bde45d79155.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/v-electrolux/extractcookie"] = plugin{
		Create: p6d77d5d507f5344a.CreateConfig,
		New:    p6d77d5d507f5344a.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/v-electrolux/http2grpc"] = plugin{
		Create: p58e152f0f7bf56bc.CreateConfig,
		New:    p58e152f0f7bf56bc.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/v-electrolux/tlsclientcertforward"] = plugin{
		Create: p5eb07795cc09a201.CreateConfig,
		New:    p5eb07795cc09a201.New,
		Version: "v1.0.2",
	}

	pluginMap["github.com/valebedeva/convertheader"] = plugin{
		Create: p6b67549cfe479764.CreateConfig,
		New:    p6b67549cfe479764.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/valksor/traefik-conditional-headers"] = plugin{
		Create: p4026bc342965301.CreateConfig,
		New:    p4026bc342965301.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/VanagaS/charset-converter"] = plugin{
		Create: p11dbe61bccca063.CreateConfig,
		New:    p11dbe61bccca063.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/VanagaS/preflight-custom-headers"] = plugin{
		Create: p790bbe560b3730b.CreateConfig,
		New:    p790bbe560b3730b.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/Vandebron/traefik-cloudflare-plugin"] = plugin{
		Create: p9f9432d22da23bac.CreateConfig,
		New:    p9f9432d22da23bac.New,
		Version: "v1.0.1",
	}

	pluginMap["github.com/Vandebron/traefik-keycloak"] = plugin{
		Create: pf8867bec0324b389.CreateConfig,
		New:    pf8867bec0324b389.New,
		Version: "v0.0.7",
	}

	pluginMap["github.com/vaspapadopoulos/traefik-cookie-handler-plugin"] = plugin{
		Create: p5d2fbb210f7239e3.CreateConfig,
		New:    p5d2fbb210f7239e3.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/vercel-saleseng/traefik-oidc-auth-plugin"] = plugin{
		Create: pdd53d567a4695233.CreateConfig,
		New:    pdd53d567a4695233.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/vidiemme/accesscontrol-ip-or-header"] = plugin{
		Create: p38be0987ed87a123.CreateConfig,
		New:    p38be0987ed87a123.New,
		Version: "v0.1.4",
	}

	pluginMap["github.com/vidosits/header-pattern-proxy"] = plugin{
		Create: p55c3c0b1fb6420c8.CreateConfig,
		New:    p55c3c0b1fb6420c8.New,
		Version: "v1.2.0",
	}

	pluginMap["github.com/vincentinttsh/cloudflareip"] = plugin{
		Create: p15cd59cddf3e4a29.CreateConfig,
		New:    p15cd59cddf3e4a29.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/vincentinttsh/rewriteheaders"] = plugin{
		Create: pe576660eb45c4da1.CreateConfig,
		New:    pe576660eb45c4da1.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/virtualzone/rewriteheaders"] = plugin{
		Create: pb66a9da48dc1ff18.CreateConfig,
		New:    pb66a9da48dc1ff18.New,
		Version: "v0.2.0",
	}

	pluginMap["github.com/vitaly-erofeev/avanpost_jwt_modification"] = plugin{
		Create: pf418760a6fdf67b3.CreateConfig,
		New:    pf418760a6fdf67b3.New,
		Version: "v0.2.3",
	}

	pluginMap["github.com/vnghia/traefik-plugin-rewrite-cookie-path"] = plugin{
		Create: pa5f6a5675f4d8a1.CreateConfig,
		New:    pa5f6a5675f4d8a1.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/vslinko/secret-auth"] = plugin{
		Create: pd068c74d517a2596.CreateConfig,
		New:    pd068c74d517a2596.New,
		Version: "v0.1.3",
	}

	pluginMap["github.com/vtacquet/redbase-plugin"] = plugin{
		Create: p806c0df7c2d4c279.CreateConfig,
		New:    p806c0df7c2d4c279.New,
		Version: "v0.1.6",
	}

	pluginMap["github.com/Wafris/wafris-traefik"] = plugin{
		Create: p2152c0f1d7208d4a.CreateConfig,
		New:    p2152c0f1d7208d4a.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/WagnerPMC/reverseguard"] = plugin{
		Create: p2c86845eef676226.CreateConfig,
		New:    p2c86845eef676226.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/WalterP/traefik-mtls-check-plugin"] = plugin{
		Create: p5a5faa7efbd63314.CreateConfig,
		New:    p5a5faa7efbd63314.New,
		Version: "v0.1.1",
	}

	pluginMap["github.com/wbpaygate/traefik-headers"] = plugin{
		Create: pdfcc4f318391e065.CreateConfig,
		New:    pdfcc4f318391e065.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/wbpaygate/traefik-ratelimit"] = plugin{
		Create: p1c307dd9120b21a1.CreateConfig,
		New:    p1c307dd9120b21a1.New,
		Version: "v0.0.15",
	}

	pluginMap["github.com/wdonne/traefikoidc"] = plugin{
		Create: pedb7dfcb234a7775.CreateConfig,
		New:    pedb7dfcb234a7775.New,
		Version: "v1.2.10",
	}

	pluginMap["github.com/wiltonsr/ldapAuth"] = plugin{
		Create: p464926337b0895a5.CreateConfig,
		New:    p464926337b0895a5.New,
		Version: "v0.1.11",
	}

	pluginMap["github.com/WithourAI/path-auth-redirector"] = plugin{
		Create: p51aeb3800fffa746.CreateConfig,
		New:    p51aeb3800fffa746.New,
		Version: "v0.0.7",
	}

	pluginMap["github.com/worldline-go/traefik-plugin-hello"] = plugin{
		Create: pa2008ad345e09978.CreateConfig,
		New:    pa2008ad345e09978.New,
		Version: "v0.1.0",
	}

	pluginMap["github.com/wzator/headerblock"] = plugin{
		Create: p851882b77bc0c090.CreateConfig,
		New:    p851882b77bc0c090.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/x-ream/traefik-plugin-jwt-antpath"] = plugin{
		Create: p69b1d9e8d5ce7e56.CreateConfig,
		New:    p69b1d9e8d5ce7e56.New,
		Version: "v0.2.2",
	}

	pluginMap["github.com/xabinapal/traefik-authentik-forward-plugin"] = plugin{
		Create: pf6a25e65d1514146.CreateConfig,
		New:    pf6a25e65d1514146.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/xabinapal/traefik-customizable-auth-forward-plugin"] = plugin{
		Create: pe39f61d858c0d73b.CreateConfig,
		New:    pe39f61d858c0d73b.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/XciD/traefik-plugin-rewrite-headers"] = plugin{
		Create: p4322cf273aeb6827.CreateConfig,
		New:    p4322cf273aeb6827.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/xethlyx/traefik-real-ip"] = plugin{
		Create: pbe14feca14520c32.CreateConfig,
		New:    pbe14feca14520c32.New,
		Version: "v1.0.7",
	}

	pluginMap["github.com/xmd3/traefik-cf-ip"] = plugin{
		Create: p32f8cd28c37ba820.CreateConfig,
		New:    p32f8cd28c37ba820.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/Yeicor/traefikgothauth"] = plugin{
		Create: pce56639e32815d93.CreateConfig,
		New:    pce56639e32815d93.New,
		Version: "v0.4.10",
	}

	pluginMap["github.com/Yeicor/traefikoidc"] = plugin{
		Create: pe3b0133900ee20d4.CreateConfig,
		New:    pe3b0133900ee20d4.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/yoeluk/traefik-authz-plugin"] = plugin{
		Create: p3c2585221ef448d3.CreateConfig,
		New:    p3c2585221ef448d3.New,
		Version: "v0.4.2",
	}

	pluginMap["github.com/yurasavin/traefiktimestampheader"] = plugin{
		Create: pdc262956775441bd.CreateConfig,
		New:    pdc262956775441bd.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/zackzackzackzack/traefik_datadog_tracing"] = plugin{
		Create: p5886b671fae37a5a.CreateConfig,
		New:    p5886b671fae37a5a.New,
		Version: "v0.0.1",
	}

	pluginMap["github.com/zalbiraw/custommetrics"] = plugin{
		Create: p4969a4947a45f2c8.CreateConfig,
		New:    p4969a4947a45f2c8.New,
		Version: "v0.0.5",
	}

	pluginMap["github.com/zalbiraw/formdata"] = plugin{
		Create: p1b823c529776d33b.CreateConfig,
		New:    p1b823c529776d33b.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/zalbiraw/headertoquery"] = plugin{
		Create: p79a3c3471f6df16d.CreateConfig,
		New:    p79a3c3471f6df16d.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/zalbiraw/jwtvalidator"] = plugin{
		Create: pe0aed7bf48d556a4.CreateConfig,
		New:    pe0aed7bf48d556a4.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/zalbiraw/ociaitoopenai"] = plugin{
		Create: p978e455bacce01f0.CreateConfig,
		New:    p978e455bacce01f0.New,
		Version: "v0.2.3",
	}

	pluginMap["github.com/zalbiraw/ociauth"] = plugin{
		Create: p342d93e806e33745.CreateConfig,
		New:    p342d93e806e33745.New,
		Version: "v0.1.2",
	}

	pluginMap["github.com/zalbiraw/pcprovider"] = plugin{
		Create: p8a9943efa7058db0.CreateConfig,
		New:    p8a9943efa7058db0.New,
		Version: "v0.0.7",
	}

	pluginMap["github.com/zalbiraw/requesttemplate"] = plugin{
		Create: p6a0e4bc5fbbcb72.CreateConfig,
		New:    p6a0e4bc5fbbcb72.New,
		Version: "v0.0.4",
	}

	pluginMap["github.com/zalbiraw/tokencounter"] = plugin{
		Create: p30e75bbc7aebb56d.CreateConfig,
		New:    p30e75bbc7aebb56d.New,
		Version: "v0.0.16",
	}

	pluginMap["github.com/zalbiraw/traefikprovider"] = plugin{
		Create: p6226bc26ee729ddc.CreateConfig,
		New:    p6226bc26ee729ddc.New,
		Version: "v0.0.2",
	}

	pluginMap["github.com/zekihan/cloudflarewarp"] = plugin{
		Create: pae98f24de4ec16b6.CreateConfig,
		New:    pae98f24de4ec16b6.New,
		Version: "v1.4.1",
	}

	pluginMap["github.com/zekihan/traefik-rate-limit"] = plugin{
		Create: p8fa41ff278640d29.CreateConfig,
		New:    p8fa41ff278640d29.New,
		Version: "v0.2.1",
	}

	pluginMap["github.com/zekihan/traefik-real-ip"] = plugin{
		Create: pa69bd68632a6d4a.CreateConfig,
		New:    pa69bd68632a6d4a.New,
		Version: "v0.1.12",
	}

	pluginMap["github.com/ZeroGachis/traefik-auth-middleware"] = plugin{
		Create: p425bfe4163da80fe.CreateConfig,
		New:    p425bfe4163da80fe.New,
		Version: "v1.1.0",
	}

	pluginMap["github.com/ZeroGachis/traefik-block-terminated-clients"] = plugin{
		Create: p73ea40d9c940ae52.CreateConfig,
		New:    p73ea40d9c940ae52.New,
		Version: "v0.3.0",
	}

	pluginMap["github.com/ZeroGachis/traefik-magic-jwt"] = plugin{
		Create: pa6c334d53ef87999.CreateConfig,
		New:    pa6c334d53ef87999.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/ZeroGachis/traefik-oauth"] = plugin{
		Create: pe84af6ccaf83c1b6.CreateConfig,
		New:    pe84af6ccaf83c1b6.New,
		Version: "v0.3.1",
	}

	pluginMap["github.com/ZeroGachis/traefik-request-id"] = plugin{
		Create: p6d2f7cca31cb119c.CreateConfig,
		New:    p6d2f7cca31cb119c.New,
		Version: "v0.1.5",
	}

	pluginMap["github.com/zhaohongyang0701/add-trace-response-header"] = plugin{
		Create: p653a7a3120d40757.CreateConfig,
		New:    p653a7a3120d40757.New,
		Version: "v1.0.5",
	}

	pluginMap["github.com/zhaohongyang0701/scanblock"] = plugin{
		Create: pdbbb9395da8784b4.CreateConfig,
		New:    pdbbb9395da8784b4.New,
		Version: "v1.1.0",
	}

	pluginMap["github.com/zhaohongyang0701/trace"] = plugin{
		Create: p4b1d3bc0d1dd8d0a.CreateConfig,
		New:    p4b1d3bc0d1dd8d0a.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/zorgzerg/traefik-s3-proxy-plugin"] = plugin{
		Create: pcb5a5763c003f300.CreateConfig,
		New:    pcb5a5763c003f300.New,
		Version: "v1.0.0",
	}

	pluginMap["github.com/ztelliot/traefik-echoserver"] = plugin{
		Create: pd38a079e07f408ad.CreateConfig,
		New:    pd38a079e07f408ad.New,
		Version: "v0.1.5",
	}

	pluginMap["github.com/zyeming/rejectcontries"] = plugin{
		Create: p911d7a9db819e335.CreateConfig,
		New:    p911d7a9db819e335.New,
		Version: "v0.0.3",
	}

}

var pluginMap = map[string]plugin{}

func NewPlugin(ctx context.Context, name string, config string, next http.Handler) (http.Handler, error) {
	c := reflect.ValueOf(pluginMap[name].Create).Call([]reflect.Value{})[0].Interface()

	err := json.Unmarshal([]byte(config), &c)
	if err != nil {
		return nil, err
	}

	results := reflect.ValueOf(pluginMap[name].New).Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(next),
		reflect.ValueOf(c),
		reflect.ValueOf("name"),
	})

	var h http.Handler
	if !results[0].IsNil() {
		h = results[0].Interface().(http.Handler)
	}
	if !results[1].IsNil() {
		err := results[1].Interface().(error)
		if err != nil {
			return nil, err
		}
	}

	return h, nil
}

func Plugins() []string {
	var plugins []string
	for name, plugin := range pluginMap {
		plugins = append(plugins, name+"@"+plugin.Version)
	}

	return plugins
}
