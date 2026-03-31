defmodule KnirvArenaWeb.Router do
  use KnirvArenaWeb, :router

  pipeline :api do
    plug :accepts, ["json"]
  end

  scope "/api", KnirvArenaWeb do
    pipe_through :api
  end
end
