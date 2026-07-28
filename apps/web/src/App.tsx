import { QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { BrowserRouter } from "react-router-dom";
import { createQueryClient } from "./lib/queryClient";
import { AuthProvider } from "./context/AuthContext";
import { AppRouter } from "./router";
import { Layout } from "./layout/Layout";

const queryClient = createQueryClient();

export function App(): React.ReactNode {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter>
          <Layout>
            <AppRouter />
          </Layout>
        </BrowserRouter>
      </AuthProvider>
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  );
}
