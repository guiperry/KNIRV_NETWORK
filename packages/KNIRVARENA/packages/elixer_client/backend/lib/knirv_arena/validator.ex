defmodule KnirvArena.Validator do
  @socket_path "/var/run/knirv/arena_out.sock"
  require Logger

  def request_validation(error_id, trajectory) do
    payload = %{
      type: "VALIDATE_SUBMISSION",
      error_id: error_id,
      trajectory: trajectory,
      timestamp: DateTime.utc_now()
    } |> Jason.encode!()

    # Connect to the Go server's UDS listener
    case :gen_tcp.connect({:local, @socket_path}, 0, [:binary, active: false]) do
      {:ok, socket} ->
        :gen_tcp.send(socket, payload)
        # Wait for the Go server to run the solution in the sandbox
        case :gen_tcp.recv(socket, 0, 10_000) do
          {:ok, response} -> handle_validation_response(response)
          {:error, :timeout} -> {:error, :validation_timeout}
        end
      {:error, reason} -> {:error, reason}
    end
  end

  defp handle_validation_response(data) do
    case Jason.decode(data) do
      {:ok, %{"status" => "SUCCESS", "merkle_proof" => proof}} -> {:ok, proof}
      {:ok, %{"status" => "FAIL", "reason" => reason}} -> {:error, reason}
      _ -> {:error, :malformed_response}
    end
  end
end