server "api" {
  error_file = "./../server_error.html"

  api {
    error_file = "./../api_error.json"

    endpoint "/{path}/{hostname}/{origin}" {
      path = "/set/by/endpoint/unset/by/backend"
      proxy {
        backend "anything" {
          path = "/anything"
          origin = "http://${request.path_params.origin}"
          hostname = request.path_params.hostname
          set_response_headers = {
            x-origin = request.path_params.origin
          }
        }
      }
    }
  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    path = "/not-found-anything"
    origin = env.COUPER_TEST_BACKEND_ADDR
  }
}
