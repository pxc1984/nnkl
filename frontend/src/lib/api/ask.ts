import { api } from "$lib/api/client";
import { getStoredAuthSession } from "$lib/auth/storage";
import { API_URL } from "$lib/config";

export type AskRequest = {
  query: string;
  mode?: "naive" | "local" | "global" | "hybrid";
};

export type Reference = {
  id: string;
  filename: string;
  type: string;
  createdAt: string;
};

export type AskResponse = {
  answer: string;
  mode: string;
  sessionId?: string;
  references?: Reference[];
};

export type QuerySessionResponse = {
  id: string;
  query: string;
  answer: string;
  mode: string;
  createdAt: string;
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

export async function listQuerySessions(
  pageSize = 8,
): Promise<QuerySessionResponse[]> {
  const response = await api.get<QuerySessionResponse[]>(
    "/api/v1/data/ask/sessions",
    {
      params: { pageSize },
    },
  );
  return response.data;
}

export async function getQuerySession(
  sessionId: string,
): Promise<QuerySessionResponse> {
  const response = await api.get<QuerySessionResponse>(
    `/api/v1/data/ask/session/${sessionId}`
  );
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

  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${(await response.text()).trim()}`);
  }

  const reader = response.body?.getReader();
  if (!reader) {
    return;
  }

  const decoder = new TextDecoder();
  let buffer = "";

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");

      // Keep the last incomplete line in the buffer
      buffer = lines.pop() ?? "";

      for (const line of lines) {
        if (line.trim() === "") {
          continue;
        }

        try {
          const chunk = JSON.parse(line) as StreamMessage;
          if (chunk.error) {
            throw new Error(chunk.error);
          }
          if (chunk.response) {
            onChunk(chunk.response);
          }
        } catch (e) {
          console.error("Failed to parse stream chunk:", e);
        }
      }
    }

    // Process any remaining buffer
    if (buffer.trim() !== "") {
      try {
        const chunk = JSON.parse(buffer) as StreamMessage;
        if (chunk.error) {
          throw new Error(chunk.error);
        }
        if (chunk.response) {
          onChunk(chunk.response);
        }
      } catch (e) {
        console.error("Failed to parse final stream chunk:", e);
      }
    }
  } finally {
    reader.releaseLock();
  }
}