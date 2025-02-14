server "api" {
  error_file = "./../integration/server_error.html"

  endpoint "/pdf" {
    request "pdf" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/pdf"
    }

    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/pdf"
    }

    response {
      headers = {
        x-body = backend_responses.default.body
      }
      body = backend_responses.pdf.body
    }
  }

  endpoint "/post" {
    request "a" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      body = request.body
    }

    request "b" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      body = request.body
    }

    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      set_request_headers = {
        x-body = request.body
      }
    }
  }
}
