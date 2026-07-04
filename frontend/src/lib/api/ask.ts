import { api } from "$lib/api/client";

export type AskRequest = {
  query: string;
  mode?: "naive" | "local" | "global" | "hybrid";
};

export type AskResponse = {
  answer: string;
  mode: string;
};

export async function askQuestion(
  query: string,
  mode: AskRequest["mode"] = "hybrid",
): Promise<AskResponse> {
  const response = await api.post<AskResponse>("/api/v1/data/ask", {
    query,
    mode,
  });
  return response.data;
}
