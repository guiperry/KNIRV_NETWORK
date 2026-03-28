defmodule Knirvarena.MixProject do
  use Mix.Project

  def project do
    [
      app: :knirvarena,
      version: "0.1.0",
      elixir: "~> 1.12",
      start_permanent: Mix.env() == :prod,
      deps: deps()
    ]
  end

  # Run "mix help compile.app" to learn about applications.
  def application do
    [
      extra_applications: [:logger],
      mod: {Knirvarena.Application, []},
      # Start Phoenix PubSub for distributed messaging
      env: [pubsub_server: true]
    ]
  end

  # Run "mix help deps" to learn about dependencies.
  defp deps do
    [
      # JSON encoding/decoding
      {:jason, "~> 1.2"},
      
      # Phoenix PubSub for distributed messaging (compatible with Phoenix 1.4)
      {:phoenix_pubsub, "~> 1.1"},
      
      # Phoenix HTML for templates (if needed)
      {:phoenix_html, "~> 3.0"},
      
      # Plug.Parsers for parsing request bodies
      {:plug_cowboy, "~> 2.0"},
      
      # Poison for JSON parsing (used by Plug.Parsers)
      {:poison, "~> 3.1"},
      
      # Use Phoenix 1.4 which is compatible with Elixir 1.12.2
      {:phoenix, "1.4.18"},
      
      # Cowboy for HTTP server
      {:cowboy, "~> 2.9"}
    ]
  end
end
