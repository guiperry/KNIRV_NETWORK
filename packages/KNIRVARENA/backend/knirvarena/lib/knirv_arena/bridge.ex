defmodule KnirvArena.Bridge do
  @socket_path "/var/run/knirv/arena_out.sock"
  require Logger

  def get_reputation(user_id) do
    payload = %{
      type: "GET_REPUTATION",
      user_id: user_id,
      timestamp: DateTime.utc_now()
    } |> Jason.encode!()

    # Connect to the Go server's UDS listener
    case :gen_tcp.connect({:local, @socket_path}, 0, [:binary, active: false]) do
      {:ok, socket} ->
        :gen_tcp.send(socket, payload)
        # Wait for the Go server to respond with reputation data
        case :gen_tcp.recv(socket, 0, 10_000) do
          {:ok, response} -> handle_reputation_response(response)
          {:error, :timeout} -> {:error, :request_timeout}
        end
      {:error, reason} -> {:error, reason}
    end
  end

  defp handle_reputation_response(data) do
    case Jason.decode(data) do
      {:ok, %{"status" => "SUCCESS", "reputation" => score}} when is_number(score) ->
        {:ok, score}
      {:ok, %{"status" => "ERROR", "reason" => reason}} ->
        {:error, reason}
      _ -> {:error, :malformed_response}
    end
  end
end