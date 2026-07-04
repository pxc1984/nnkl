import { api } from "$lib/api/client";
import { getStoredAuthSession } from "$lib/auth/storage";
import { API_URL } from "$lib/config";

export type AskRequest = {
  query: string;
  mode?: "naive" | "local" | "global" | "hybrid";
};

export type AskResponse = {
  answer: string;
  mode: string;
  sessionId: string;
};

const NO_CONTEXT_MARKER = "[no-context]";

export function isNoContextAnswer(answer: string): boolean {
  return answer.trim().includes(NO_CONTEXT_MARKER);
}

export async function askQuestion(
  query: string,
  mode: AskRequest["mode"] = "naive",
): Promise<AskResponse> {
  const response = await api.post<AskResponse>("/api/v1/data/ask", {
    query,
    mode,
  });
  return response.data;
}

type StreamMessage = {
  response?: string;
  error?: string;
};

export async function streamQuestion(
  query: string,
  mode: AskRequest["mode"] = "naive",
  onChunk: (chunk: string) => void,
): Promise<void> {
  const session = getStoredAuthSession();
  const response = await fetch(`${API_URL}/api/v1/data/ask/stream`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(session?.accessToken
        ? { Authorization: `Bearer ${session.accessToken}` }
        : {}),
    },
    body: JSON.stringify({ query, mode }),
  });

  if (!response.ok || !response.body) {
    throw new Error(`Knowledge base returned ${response.status}`);
  }

  const reader = response.body.pipeThrough(new TextDecoderStream()).getReader();
  let pending = "";
  while (true) {
    const { value, done } = await reader.read();
    pending += value ?? "";
    const lines = pending.split("\n");
    pending = lines.pop() ?? "";
    for (const line of lines) {
      if (!line.trim()) continue;
      const message = JSON.parse(line) as StreamMessage;
      if (message.error) throw new Error(message.error);
      if (message.response) onChunk(message.response);
    }
    if (done) break;
  }

  if (pending.trim()) {
    const message = JSON.parse(pending) as StreamMessage;
    if (message.error) throw new Error(message.error);
    if (message.response) onChunk(message.response);
  }
}
