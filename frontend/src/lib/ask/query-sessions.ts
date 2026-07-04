import { writable } from "svelte/store";

export type SidebarQuerySession = {
  id: string;
  name: string;
  preview: string;
  time: string;
  active?: boolean;
};

const MAX_QUERY_SESSIONS = 8;

export const querySessions = writable<SidebarQuerySession[]>([]);

export function prependQuerySession(session: {
  id: string;
  query: string;
  answer: string;
}): void {
  querySessions.update((items) => {
    const nextItem: SidebarQuerySession = {
      id: session.id,
      name: truncate(session.query.trim(), 72),
      preview: truncate(stripMarkdown(session.answer), 120),
      time: "Сейчас",
      active: true,
    };

    return [
      nextItem,
      ...items
        .filter((item) => item.id !== session.id)
        .map((item) => ({ ...item, active: false })),
    ].slice(0, MAX_QUERY_SESSIONS);
  });
}

function truncate(value: string, maxLength: number): string {
  const normalized = value.replace(/\s+/g, " ").trim();
  if (normalized.length <= maxLength) {
    return normalized;
  }

  return `${normalized.slice(0, maxLength - 1).trimEnd()}...`;
}

function stripMarkdown(value: string): string {
  return value
    .replace(/```[\s\S]*?```/g, " ")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/!\[[^\]]*\]\([^)]*\)/g, " ")
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/^[>#*-]\s+/gm, "")
    .replace(/[*_~#]/g, "")
    .replace(/\s+/g, " ")
    .trim();
}
