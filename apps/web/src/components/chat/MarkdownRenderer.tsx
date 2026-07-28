import { memo, type ReactNode } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { Components } from "react-markdown";
import { CodeBlock } from "./CodeBlock";

function extractText(node: ReactNode): string {
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(extractText).join("");
  return "";
}

interface MarkdownRendererProps {
  readonly content: string;
}

const components: Components = {
  code({ className, children, ...props }) {
    const match = /language-(\w+)/.exec(className ?? "");
    const code = extractText(children).replace(/\n$/, "");

    if (match !== null) {
      return <CodeBlock language={match[1]} code={code} />;
    }

    return (
      <code
        className="rounded bg-slate-800 px-1.5 py-0.5 text-sm text-slate-200"
        {...props}
      >
        {children}
      </code>
    );
  },
  pre({ children }) {
    return <>{children}</>;
  },
  table({ children }) {
    return (
      <div className="my-3 overflow-x-auto">
        <table className="w-full border-collapse text-sm">{children}</table>
      </div>
    );
  },
  th({ children }) {
    return (
      <th className="border border-slate-700 bg-slate-800 px-3 py-2 text-left font-medium text-slate-200">
        {children}
      </th>
    );
  },
  td({ children }) {
    return (
      <td className="border border-slate-700 px-3 py-2 text-slate-300">
        {children}
      </td>
    );
  },
  a({ children, href }) {
    return (
      <a
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        className="text-forge-400 underline hover:text-forge-300"
      >
        {children}
      </a>
    );
  },
  ul({ children }) {
    return <ul className="my-2 list-disc pl-6 text-slate-300">{children}</ul>;
  },
  ol({ children }) {
    return <ol className="my-2 list-decimal pl-6 text-slate-300">{children}</ol>;
  },
  li({ children }) {
    return <li className="my-1">{children}</li>;
  },
  p({ children }) {
    return <p className="my-2 leading-relaxed text-slate-300">{children}</p>;
  },
  h1({ children }) {
    return <h1 className="my-4 text-xl font-bold text-white">{children}</h1>;
  },
  h2({ children }) {
    return <h2 className="my-3 text-lg font-bold text-white">{children}</h2>;
  },
  h3({ children }) {
    return (
      <h3 className="my-2 text-base font-semibold text-white">{children}</h3>
    );
  },
  blockquote({ children }) {
    return (
      <blockquote className="my-3 border-l-4 border-forge-500 bg-slate-800/50 py-2 pl-4 italic text-slate-400">
        {children}
      </blockquote>
    );
  },
  hr() {
    return <hr className="my-6 border-slate-700" />;
  },
};

export const MarkdownRenderer = memo(function MarkdownRenderer({
  content,
}: MarkdownRendererProps): React.ReactNode {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={components}
    >
      {content}
    </ReactMarkdown>
  );
});
