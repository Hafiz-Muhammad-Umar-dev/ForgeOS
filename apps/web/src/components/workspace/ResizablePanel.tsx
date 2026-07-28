import { useCallback, useRef, useState } from "react";

interface ResizablePanelProps {
  readonly children: React.ReactNode;
  readonly defaultWidth?: number;
  readonly minWidth?: number;
  readonly maxWidth?: number;
  readonly side?: "left" | "right";
}

export function ResizablePanel({
  children,
  defaultWidth = 280,
  minWidth = 180,
  maxWidth = 500,
  side = "left",
}: ResizablePanelProps): React.ReactNode {
  const [width, setWidth] = useState(defaultWidth);
  const isDraggingRef = useRef(false);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      isDraggingRef.current = true;

      const handleMouseMove = (moveEvent: MouseEvent) => {
        if (!isDraggingRef.current) return;
        const newWidth = side === "left"
          ? moveEvent.clientX
          : window.innerWidth - moveEvent.clientX;
        setWidth(Math.max(minWidth, Math.min(maxWidth, newWidth)));
      };

      const handleMouseUp = () => {
        isDraggingRef.current = false;
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
      };

      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    },
    [side, minWidth, maxWidth],
  );

  return (
    <div className="relative shrink-0" style={{ width: `${String(width)}px` }}>
      {children}
      <div
        onMouseDown={handleMouseDown}
        className={`absolute top-0 ${
          side === "left" ? "right-0 cursor-col-resize" : "left-0 cursor-col-resize"
        } h-full w-1 transition-colors hover:bg-forge-500/50 active:bg-forge-500`}
      />
    </div>
  );
}
