import type { StreamChunk } from "../../types/chat";

export class StreamParser {
  private buffer = "";

  public append(data: string): StreamChunk[] {
    this.buffer += data;
    const chunks: StreamChunk[] = [];
    const lines = this.buffer.split("\n");
    this.buffer = lines.pop() ?? "";

    for (const line of lines) {
      if (line.startsWith("data: ")) {
        const payload = line.slice(6).trim();
        if (payload === "[DONE]") {
          chunks.push({ content: "", done: true });
        } else {
          try {
            const parsed = JSON.parse(payload) as {
              content?: string;
              error?: string;
            };
            chunks.push({
              content: parsed.content ?? "",
              done: false,
              error: parsed.error,
            });
          } catch {
            chunks.push({ content: payload, done: false });
          }
        }
      }
    }
    return chunks;
  }

  public reset(): void {
    this.buffer = "";
  }
}
