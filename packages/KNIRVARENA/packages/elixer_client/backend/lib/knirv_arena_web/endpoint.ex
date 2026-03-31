defmodule KnirvArenaWeb.Endpoint do
  use Phoenix.Endpoint, otp_app: :knirvarena

  socket "/socket", KnirvArenaWeb.UserSocket,
    websocket: true,
    longpoll: false

  import Plug.Conn

  # Serve static files
  plug Plug.Static,
    at: "/",
    from: :knirv_arena,
    gzip: false,
    only: ~w(assets fonts images js favicon.ico robots.txt)

  # Code reloading
  if Mix.env() in [:dev, :test] do
    plug Phoenix.CodeReloader
  end

  # Logging
  plug Plug.Logger

  # Parse request body
  plug Plug.Parsers,
    parsers: [:urlencoded, :multipart, :json],
    pass: ["*/*"],
    json_decoder: Poison

  # Session management
  plug Plug.Session,
    store: :cookie,
    key: "_knirvarena_key",
    signing_salt: "KnirvArenaSessionSalt"

  # Router
  plug KnirvArenaWeb.Router

  # For Phoenix 1.4, we need to define a few more things
  @app_name :knirvarena

  def child_spec(_opts) do
    Plug.Adapters.Cowboy.child_spec(scheme: :http, plug: KnirvArenaWeb.Endpoint, options: [port: 4000])
  end
end