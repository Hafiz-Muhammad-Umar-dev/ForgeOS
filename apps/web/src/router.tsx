import { Route, Routes } from "react-router-dom";
import { ProtectedRoute } from "./components/ProtectedRoute";
import { Dashboard } from "./pages/Dashboard";
import { IntentView } from "./pages/IntentView";
import { TaskBoard } from "./pages/TaskBoard";
import { WorkspaceView } from "./pages/WorkspaceView";
import { DeploymentView } from "./pages/DeploymentView";
import { Login } from "./pages/Login";

export function AppRouter(): React.ReactNode {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <Dashboard />
          </ProtectedRoute>
        }
      />
      <Route
        path="/intents"
        element={
          <ProtectedRoute>
            <IntentView />
          </ProtectedRoute>
        }
      />
      <Route
        path="/tasks"
        element={
          <ProtectedRoute>
            <TaskBoard />
          </ProtectedRoute>
        }
      />
      <Route
        path="/workspaces"
        element={
          <ProtectedRoute>
            <WorkspaceView />
          </ProtectedRoute>
        }
      />
      <Route
        path="/deployments"
        element={
          <ProtectedRoute>
            <DeploymentView />
          </ProtectedRoute>
        }
      />
    </Routes>
  );
}
