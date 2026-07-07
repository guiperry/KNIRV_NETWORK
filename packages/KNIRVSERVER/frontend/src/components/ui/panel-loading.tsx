import { cn } from "@/lib/utils"
import { Skeleton } from "./skeleton"

interface PanelLoadingProps {
  message?: string
  variant?: "spinner" | "skeleton" | "minimal"
  rows?: number
  className?: string
}

function PanelLoading({ message = "Loading...", variant = "spinner", rows = 3, className }: PanelLoadingProps) {
  if (variant === "minimal") {
    return (
      <div data-slot="panel-loading" className={cn("flex items-center gap-2 text-muted-foreground text-sm", className)}>
        <span className="inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
        <span>{message}</span>
      </div>
    )
  }

  if (variant === "skeleton") {
    return (
      <div data-slot="panel-loading" className={cn("flex flex-col gap-3 p-4", className)}>
        <Skeleton className="h-5 w-48" />
        {Array.from({ length: rows }, (_, i) => (
          <Skeleton key={i} className={cn("h-4", i === rows - 1 ? "w-3/4" : "w-full")} />
        ))}
      </div>
    )
  }

  return (
    <div data-slot="panel-loading" className={cn("flex flex-col items-center justify-center gap-3 p-8", className)}>
      <div className="relative h-10 w-10">
        <div className="absolute inset-0 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
      <p className="text-muted-foreground text-sm">{message}</p>
    </div>
  )
}

export { PanelLoading }
