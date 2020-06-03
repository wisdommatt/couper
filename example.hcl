frontend "couperConnect" {
    base_path = "/api/v1/"

    files {
        document_root = "./public"
    }

    endpoint "/proxy/" {
        backend "Proxy" {
            description = "optional field"
            origin_address = "couper.io:${442 + 1}"
            origin_host = "couper.io"
            request {
                headers = {
                    X-My-Custom-Foo-UA = ["ua:${req.headers.User-Agent}", "muh"]
                    X-Env-User = ["${env.USER}"]
                }
            }

            response {
                headers = {
                    Server = ["mySuperService"]
                }
            }
        }
    }

    endpoint "/httpbin/" {
        backend "Proxy" {
            path = "/headers/" #Optional and only if set, remove basePath+endpoint path
            description = "optional field"
            origin_address = "httpbin.org:443"
            origin_host = "httpbin.org"
            request {
                headers = {
                    X-Env-User = ["${env.USER}"]
                    X-Req-Header = ["${req.headers.X-Set-Me}"]
                }
            }

            response {
                # TODO: optional block's ?
            }
        }
    }   
}
