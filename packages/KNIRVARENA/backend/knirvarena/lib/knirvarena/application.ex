defmodule Knirvarena.Application do
  # See https://hexdocs.pm/elixir/Application.html
  # for more information on OTP Applications
  @moduledoc false

  use Application

  @impl true
  def start(_type, _args) do
    children = [
      # Start the PubSub system (Phoenix 1.4 / PubSub 1.1 style)
      %{
        id: Phoenix.PubSub.PG2,
        start: {Phoenix.PubSub.PG2, :start_link, [KnirvArena.PubSub, []]}
      },

      # Start the Registry for tracking ResolutionSession GenServers
      {Registry, [keys: :unique, name: KnirvArena.SessionRegistry]},
      
      # Start the Supervisor for managing ResolutionSession GenServers
      {KnirvArena.ResolutionSupervisor, []},
      
      # Start the BountyListener to receive ErrorNodeBounties from KNIRVSERVER
      {KnirvArena.BountyListener, []},

      # Start the Phoenix Endpoint (using the manual child_spec defined in Endpoint)
      KnirvArenaWeb.Endpoint
    ]

    opts = [strategy: :one_for_one, name: Knirvarena.Supervisor]
    Supervisor.start_link(children, opts)
  end
end