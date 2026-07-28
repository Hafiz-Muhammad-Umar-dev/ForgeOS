import { useCallback, useState } from "react";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import { oneDark } from "react-syntax-highlighter/dist/esm/styles/prism";

interface CodeBlockProps {
  readonly language: string;
  readonly code: string;
}

export function CodeBlock({ language, code }: CodeBlockProps): React.ReactNode {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(code).catch(() => undefined);
    setCopied(true);
    setTimeout(() => {
      setCopied(false);
    }, 2000);
  }, [code]);

  return (
    <div className="group relative my-3 overflow-hidden rounded-lg border border-slate-700">
      <div className="flex items-center justify-between bg-slate-800 px-4 py-1.5">
        <span className="text-xs text-slate-400">{language}</span>
        <button
          type="button"
          onClick={handleCopy}
          className="rounded px-2 py-0.5 text-xs text-slate-400 opacity-0 transition-opacity hover:bg-slate-700 hover:text-slate-200 group-hover:opacity-100"
        >
          {copied ? "Copied!" : "Copy"}
        </button>
      </div>
      <SyntaxHighlighter
        language={language}
        style={oneDark}
        customStyle={{
          margin: 0,
          borderRadius: 0,
          fontSize: "0.8125rem",
          lineHeight: 1.5,
        }}
        showLineNumbers={code.split("\n").length > 3}
      >
        {code}
      </SyntaxHighlighter>
    </div>
  );
}
