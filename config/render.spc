connection "render" {
  plugin = "render-oss/render"

  # The Render API key used to authenticate requests.
  # Create one at https://dashboard.render.com/u/settings#api-keys
  # All requests are authenticated with `Authorization: Bearer <api_key>`.
  # This can also be set via the `RENDER_API_KEY` environment variable.
  # api_key = "rnd_AbCdEf1234567890"

  # Optional. Override the Render API base URL. Defaults to https://api.render.com/v1.
  # api_url = "https://api.render.com/v1"
}
