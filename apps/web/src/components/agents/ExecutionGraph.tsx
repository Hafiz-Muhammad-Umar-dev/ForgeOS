import { useCallback, useMemo } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  type Node,
  type Edge,
  type NodeTypes,
  useNodesState,
  useEdgesState,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { TaskNode } from "./TaskNode";
import type { ExecutionPlan } from "../../types/execution";

interface ExecutionGraphProps {
  readonly plan: ExecutionPlan | null;
}

const nodeTypes = {
  taskNode: TaskNode,
} satisfies NodeTypes;

export function ExecutionGraph({ plan }: ExecutionGraphProps): React.ReactNode {
  const initialNodes: Node[] = useMemo(
    () =>
      plan?.nodes.map((n, i) => ({
        id: n.id,
        type: "taskNode",
        position: { x: 250, y: i * 120 },
        data: { label: n.label, status: n.status, progress: n.progress, runtime: n.runtime, tokens: n.tokens, cost: n.cost },
      })) ?? [],
    [plan],
  );

  const initialEdges: Edge[] = useMemo(
    () =>
      plan?.edges.map((e) => ({
        id: e.id,
        source: e.source,
        target: e.target,
        animated: e.animated ?? true,
        style: { stroke: "#475569" },
      })) ?? [],
    [plan],
  );

  const [nodes, , onNodesChange] = useNodesState(initialNodes);
  const [edges, , onEdgesChange] = useEdgesState(initialEdges);

  const onConnect = useCallback(() => undefined, []);

  if (plan === null) {
    return (
      <div className="flex h-full items-center justify-center">
        <p className="text-sm text-slate-500">No execution plan loaded.</p>
      </div>
    );
  }

  return (
    <div className="h-full w-full">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        nodeTypes={nodeTypes}
        fitView
        attributionPosition="bottom-left"
      >
        <Background color="#1e293b" gap={20} />
        <Controls className="[&>button]:bg-slate-800 [&>button]:text-slate-300 [&>button]:border-slate-700" />
        <MiniMap
          nodeColor="#334155"
          maskColor="rgba(15, 23, 42, 0.8)"
          style={{ background: "#0f172a" }}
        />
      </ReactFlow>
    </div>
  );
}
