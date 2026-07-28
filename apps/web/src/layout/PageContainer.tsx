import type { ReactNode } from "react";

interface PageContainerProps {
  readonly title: string;
  readonly description?: string;
  readonly children: ReactNode;
}

export function PageContainer({
  title,
  description,
  children,
}: PageContainerProps): React.ReactNode {
  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <header className="mb-8">
        <h1 className="text-2xl font-bold tracking-tight text-white">{title}</h1>
        {description !== undefined && (
          <p className="mt-1 text-sm text-slate-400">{description}</p>
        )}
      </header>
      <main>{children}</main>
    </div>
  );
}
