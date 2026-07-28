import type { ReactNode } from "react";
import { Navigation } from "./Navigation";

interface LayoutProps {
  readonly children: ReactNode;
}

export function Layout({ children }: LayoutProps): React.ReactNode {
  return (
    <div className="flex min-h-screen flex-col">
      <header className="border-b border-slate-800 bg-slate-900/50 backdrop-blur-sm">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3 sm:px-6 lg:px-8">
          <a href="/" className="text-lg font-bold text-white">
            ForgeOS
          </a>
          <Navigation />
        </div>
      </header>
      <div className="flex-1">{children}</div>
    </div>
  );
}
