import type {
  DataListParams,
  KnowledgeObject,
  KnowledgeObjectDetails,
} from "$lib/data/types";

export const DEFAULT_DATA_PAGE_SIZE = 20;
export const CONTENT_PREVIEW_LENGTH = 4000;

export function parseTagsInput(value: string): string[] {
  return value
    .split(",")
    .map((tag) => tag.trim())
    .filter(Boolean);
}

export function formatBytes(size?: number): string {
  if (!size || size < 0) {
    return "-";
  }

  const units = ["B", "KB", "MB", "GB", "TB"];
  let currentSize = size;
  let unitIndex = 0;

  while (currentSize >= 1024 && unitIndex < units.length - 1) {
    currentSize /= 1024;
    unitIndex += 1;
  }

  const digits = currentSize >= 10 || unitIndex === 0 ? 0 : 1;
  return `${currentSize.toFixed(digits)} ${units[unitIndex]}`;
}

export function formatDateTime(value?: string | null): string {
  if (!value) {
    return "-";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("ru-RU", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

export function getObjectTitle(
  object: KnowledgeObject | KnowledgeObjectDetails,
): string {
  return (
    object.title?.trim() || object.originalFilename?.trim() || object.filename
  );
}

export function getObjectTypeLabel(
  object: KnowledgeObject | KnowledgeObjectDetails,
): string {
  const mimeType = object.mimeType?.trim();
  if (mimeType) {
    return mimeType;
  }

  const filename = object.originalFilename || object.filename;
  const extension = filename.split(".").pop()?.trim();
  return extension ? extension.toUpperCase() : "Файл";
}

export function buildDataSearchParams(params: DataListParams): URLSearchParams {
  const searchParams = new URLSearchParams();

  if (params.query?.trim()) {
    searchParams.set("query", params.query.trim());
  }

  if (params.type?.trim()) {
    searchParams.set("type", params.type.trim());
  }

  for (const tag of params.tags ?? []) {
    if (tag.trim()) {
      searchParams.append("tags", tag.trim());
    }
  }

  if (params.page && params.page > 1) {
    searchParams.set("page", String(params.page));
  }

  if (params.pageSize && params.pageSize !== DEFAULT_DATA_PAGE_SIZE) {
    searchParams.set("pageSize", String(params.pageSize));
  }

  return searchParams;
}

export function getTagsFromSearchParams(
  searchParams: URLSearchParams,
): string[] {
  const values = searchParams.getAll("tags");
  if (values.length > 0) {
    return values.flatMap((value) => parseTagsInput(value));
  }

  const fallback = searchParams.get("tags");
  return fallback ? parseTagsInput(fallback) : [];
}

export function getMetadataEntries(
  metadata?: Record<string, unknown>,
): Array<[string, string]> {
  if (!metadata) {
    return [];
  }

  return Object.entries(metadata)
    .map(
      ([key, value]) =>
        [key, stringifyMetadataValue(value)] as [string, string],
    )
    .filter(([, value]) => value.length > 0);
}

function stringifyMetadataValue(value: unknown): string {
  if (value == null) {
    return "";
  }

  if (typeof value === "string") {
    return value;
  }

  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }

  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

export function getContentPreview(content?: string): {
  text: string;
  truncated: boolean;
} {
  if (!content) {
    return { text: "", truncated: false };
  }

  if (content.length <= CONTENT_PREVIEW_LENGTH) {
    return { text: content, truncated: false };
  }

  return {
    text: `${content.slice(0, CONTENT_PREVIEW_LENGTH).trimEnd()}...`,
    truncated: true,
  };
}
