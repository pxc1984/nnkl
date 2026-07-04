import { writable } from "svelte/store";

import type { QuerySessionResponse } from "$lib/api/ask";

export type SidebarQuerySession = {
  id: string;
  name: string;
  preview: string;
  time: string;
  query: string;
  answer: string;
  mode?: string;
  active?: boolean;
};

const MAX_QUERY_SESSIONS = 8;

export const querySessions = writable<SidebarQuerySession[]>([]);

export function setQuerySessions(sessions: QuerySessionResponse[]): void {
  querySessions.set(
    sessions.slice(0, MAX_QUERY_SESSIONS).map((session, index) => ({
      id: session.id,
      name: truncate(session.query.trim(), 72),
      preview: truncate(stripMarkdown(session.answer), 120),
      time: formatSessionTime(session.createdAt),
      query: session.query,
      answer: session.answer,
      mode: session.mode,
      active: index === 0,
    })),
  );
}

export function prependQuerySession(session: {
  id: string;
  query: string;
  answer: string;
  mode?: string;
}): void {
  querySessions.update((items) => {
    const nextItem: SidebarQuerySession = {
      id: session.id,
      name: truncate(session.query.trim(), 72),
      preview: truncate(stripMarkdown(session.answer), 120),
      time: "Сейчас",
      query: session.query,
      answer: session.answer,
      mode: session.mode,
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

export function activateQuerySession(sessionId: string): void {
  querySessions.update((items) =>
    items.map((item) => ({
      ...item,
      active: item.id === sessionId,
    })),
  );
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
    .replace(/!\[[^\]]*]\([^)]*\)/g, " ")
    .replace(/\[([^\]]+)]\([^)]*\)/g, "$1")
    .replace(/^[>#*-]\s+/gm, "")
    .replace(/[*_~#]/g, "")
    .replace(/\s+/g, " ")
    .trim();
}

export function formatSessionTime(value: string): string {
  const createdAt = new Date(value);
  if (Number.isNaN(createdAt.getTime())) {
    return "Ранее";
  }

  const diffMs = Date.now() - createdAt.getTime();
  const diffMinutes = Math.floor(diffMs / 60000);

  if (diffMinutes < 1) {
    return "Сейчас";
  }

  if (diffMinutes < 60) {
    return `${diffMinutes} мин назад`;
  }

  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) {
    return `${diffHours} ч назад`;
  }

  const diffDays = getCalendarDayDiff(createdAt, new Date());
  if (diffDays === 1) {
    return "Вчера";
  }

  if (diffDays < 7) {
    return `${diffDays} дн назад`;
  }

  return createdAt.toLocaleDateString("ru-RU", {
    day: "2-digit",
    month: "2-digit",
  });
}

function getCalendarDayDiff(from: Date, to: Date): number {
  const fromDate = new Date(from.getFullYear(), from.getMonth(), from.getDate());
  const toDate = new Date(to.getFullYear(), to.getMonth(), to.getDate());
  return Math.round((toDate.getTime() - fromDate.getTime()) / 86400000);
}
