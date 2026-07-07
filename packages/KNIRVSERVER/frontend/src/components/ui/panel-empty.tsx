import { cn } from "@/lib/utils"
import { Button } from "./button"
import type { ReactNode } from "react"

interface PanelEmptyProps {
  icon?: ReactNode
  title: string
  description?: string
  actionLabel?: string
  onAction?: () => void
  className?: string
}

function PanelEmpty({ icon, title, description, actionLabel, onAction, className }: PanelEmptyProps) {
  return (
    <div data-slot="panel-empty" className={cn("flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed p-8 text-center", className)}>
      {icon && <div className="text-muted-foreground">{icon}</div>}
      <div>
        <h3 className="font-medium text-sm">{title}</h3>
        {description && <p className="mt-1 text-muted-foreground text-xs">{description}</p>}
      </div>
      {actionLabel && onAction && (
        <Button onClick={onAction} variant="outline" size="sm" className="cursor-pointer">
          {actionLabel}
        </Button>
      )}
    </div>
  )
}

export { PanelEmpty }
