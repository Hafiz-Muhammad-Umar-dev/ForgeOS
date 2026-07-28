import { type EdgeProps, getBezierPath } from "@xyflow/react";

export function TaskEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
}: EdgeProps): React.ReactNode {
  const [edgePath] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  return (
    <path
      id={id}
      className="!stroke-slate-600"
      d={edgePath}
      strokeWidth={2}
      fill="none"
    />
  );
}
