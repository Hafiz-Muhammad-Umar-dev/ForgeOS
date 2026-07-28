import { QueryClient } from "@tanstack/react-query";

const STALE_TIME = 30_000; // 30 seconds
const RETRY_COUNT = 2;
const GC_TIME = 5 * 60_000; // 5 minutes

export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: STALE_TIME,
        gcTime: GC_TIME,
        retry: RETRY_COUNT,
        refetchOnWindowFocus: false,
      },
    },
  });
}
