defmodule KnirvArena.AssetExporter do
  require Logger

  def format_for_lorax(session) do
    %{
      instruction: session.failure_context["instruction"],
      context: session.failure_context["environment_metadata"],
      error: %{
        output: session.failure_context["failed_output"],
        type: session.failure_context["error_class"]
      },
      solution: %{
        trajectory: session.winning_trajectory,
        final_output: session.winning_trajectory
      },
      knowledge_hash: ""
    }
    |> Jason.encode!()
  end

  def export_batch(error_ids) do
    # In a real implementation, this would fetch session data from a database or cache
    # For now, we'll simulate writing to a file
    try do
      # Create training directory if it doesn't exist
      training_dir = Path.join(["priv", "training"])
      File.mkdir_p!(training_dir)
      
      file_path = Path.join([training_dir, "latest_epoch.jsonl"])
      
      File.open!(file_path, [:append], fn file ->
        Enum.each(error_ids, fn id ->
          # In reality, we'd get the actual session data
          asset = %{
            instruction: "Sample instruction",
            context: %{},
            error: %{
              output: "Sample failed output",
              type: "SampleError"
            },
            solution: %{
              trajectory: "Sample trajectory",
              final_output: "Sample solved output"
            },
            knowledge_hash: "sample-hash"
          } |> Jason.encode!()
          IO.write(file, asset <> "\n")
        end)
      end)
      
      Logger.info("Exported knowledge assets for LoRAX training")
    rescue
      e -> Logger.error("Failed to export knowledge assets: #{inspect(e)}")
    end
  end
end