import axios, { type AxiosProgressEvent } from "axios";

import { api } from "$lib/api/client";
import { DEFAULT_DATA_PAGE_SIZE } from "$lib/data/utils";
import type { GraphData, GraphNodeType } from "$lib/data/graph";
import type {
  DataListParams,
  DataTagList,
  DataUploadParams,
  DataUploadResponse,
  KnowledgeObject,
  KnowledgeObjectDetails,
  KnowledgeObjectStatus,
  PaginatedKnowledgeObjectList,
} from "$lib/data/types";

const MAX_UPLOAD_BATCH_BYTES = 9 * 1024 * 1024;

type ListKnowledgeObjectsBackendResponse = {
  items: BackendKnowledgeObject[];
  meta: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  };
};

type BackendKnowledgeObject = {
  id: string;
  filename: string;
  type?: string;
  contentType?: string;
  tags?: string[] | null;
  sizeBytes?: number;
  createdAt: string;
  updatedAt?: string;
  status?: string;
  errorMessage?: string | null;
  metadata?: Record<string, unknown>;
  title?: string;
  sha256?: string;
  hasContent?: boolean;
  hasResult?: boolean;
  outputFormat?: string;
  language?: string | null;
};

export async function listKnowledgeObjects(
  params: DataListParams,
): Promise<PaginatedKnowledgeObjectList> {
  const response = await api.get<ListKnowledgeObjectsBackendResponse>(
    "/api/v1/data",
    {
      params: {
        page: params.page,
        pageSize: params.pageSize,
        query: params.query,
        type: params.type,
        status: params.status,
      },
    },
  );

  const meta = response.data.meta ?? {
    page: params.page ?? 1,
    pageSize: params.pageSize ?? DEFAULT_DATA_PAGE_SIZE,
    total: 0,
    totalPages: 0,
  };

  return {
    items: response.data.items.map(mapKnowledgeObject),
    meta,
  };
}

export async function listDataTags(): Promise<DataTagList> {
  const response = await api.get<DataTagList>("/api/v1/tags");
  return response.data;
}

type GraphBackendResponse = {
  nodes: Array<{
    id: string;
    label: string;
    type: string;
    description?: string;
  }>;
  edges: Array<{
    source: string;
    target: string;
    label: string;
    description?: string;
    weight?: number;
  }>;
  mode: string;
};

const GRAPH_NODE_TYPES = new Set<GraphNodeType>([
  "Material",
  "Process",
  "Equipment",
  "Property",
  "Experiment",
  "Publication",
  "Expert",
  "Facility",
  "Unknown",
]);

function toGraphNodeType(raw: string): GraphNodeType {
  const normalized = raw.charAt(0).toUpperCase() + raw.slice(1).toLowerCase();
  return GRAPH_NODE_TYPES.has(normalized as GraphNodeType)
    ? (normalized as GraphNodeType)
    : "Unknown";
}

export async function queryKnowledgeGraph(
  query: string,
  mode = "hybrid",
): Promise<GraphData> {
  const response = await api.post<GraphBackendResponse>("/api/v1/data/graph", {
    query,
    mode,
  });

  return {
    nodes: response.data.nodes.map((node) => ({
      id: node.id,
      label: node.label,
      type: toGraphNodeType(node.type),
    })),
    edges: response.data.edges.map((edge) => ({
      source: edge.source,
      target: edge.target,
      label: edge.label,
    })),
  };
}

export async function uploadKnowledgeObjects(
  files: File[],
  params: DataUploadParams,
  onUploadProgress?: (progressEvent: AxiosProgressEvent) => void,
): Promise<DataUploadResponse> {
  const batches = splitUploadFilesIntoBatches(files);
  const totalBytes = files.reduce((sum, file) => sum + file.size, 0);
  const items: DataUploadResponse["items"] = [];
  let uploadedBytes = 0;

  for (const batch of batches) {
    const batchResponse = await uploadBatchWithFallback(
      batch,
      params,
      totalBytes,
      uploadedBytes,
      onUploadProgress,
    );

    items.push(...batchResponse.items);
    uploadedBytes += getBatchSize(batch);
  }

  emitUploadProgress(onUploadProgress, totalBytes, totalBytes);

  return { items };
}

async function uploadBatchWithFallback(
  batch: File[],
  params: DataUploadParams,
  totalBytes: number,
  uploadedBytesBeforeBatch: number,
  onUploadProgress?: (progressEvent: AxiosProgressEvent) => void,
): Promise<DataUploadResponse> {
  try {
    return await postUploadBatch(batch, params, (progressEvent) => {
      emitUploadProgress(
        onUploadProgress,
        totalBytes,
        uploadedBytesBeforeBatch + normalizeUploadedBytes(progressEvent, batch),
      );
    });
  } catch (error) {
    if (!isPayloadTooLargeError(error) || batch.length === 1) {
      throw error;
    }

    const items: DataUploadResponse["items"] = [];
    let uploadedBytesWithinBatch = 0;

    for (const file of batch) {
      const response = await postUploadBatch(
        [file],
        params,
        (progressEvent) => {
          emitUploadProgress(
            onUploadProgress,
            totalBytes,
            uploadedBytesBeforeBatch +
              uploadedBytesWithinBatch +
              normalizeUploadedBytes(progressEvent, [file]),
          );
        },
      );

      items.push(...response.items);
      uploadedBytesWithinBatch += file.size;
    }

    return { items };
  }
}

async function postUploadBatch(
  files: File[],
  params: DataUploadParams,
  onUploadProgress?: (progressEvent: AxiosProgressEvent) => void,
): Promise<DataUploadResponse> {
  const formData = new FormData();

  for (const file of files) {
    formData.append("data", file);
  }

  formData.append("params", JSON.stringify(params));

  const response = await api.post<DataUploadResponse>(
    "/api/v1/data",
    formData,
    {
      onUploadProgress,
    },
  );

  return response.data;
}

function splitUploadFilesIntoBatches(files: File[]): File[][] {
  const batches: File[][] = [];
  let currentBatch: File[] = [];
  let currentBatchSize = 0;

  for (const file of files) {
    if (
      currentBatch.length > 0 &&
      currentBatchSize + file.size > MAX_UPLOAD_BATCH_BYTES
    ) {
      batches.push(currentBatch);
      currentBatch = [];
      currentBatchSize = 0;
    }

    currentBatch.push(file);
    currentBatchSize += file.size;

    if (currentBatchSize >= MAX_UPLOAD_BATCH_BYTES) {
      batches.push(currentBatch);
      currentBatch = [];
      currentBatchSize = 0;
    }
  }

  if (currentBatch.length > 0) {
    batches.push(currentBatch);
  }

  return batches;
}

function getBatchSize(files: File[]): number {
  return files.reduce((sum, file) => sum + file.size, 0);
}

function normalizeUploadedBytes(
  progressEvent: AxiosProgressEvent,
  files: File[],
): number {
  const batchSize = getBatchSize(files);
  const total = progressEvent.total ?? batchSize;
  const loaded = progressEvent.loaded ?? 0;

  if (!total || total <= 0) {
    return Math.min(loaded, batchSize);
  }

  return Math.min((loaded / total) * batchSize, batchSize);
}

function emitUploadProgress(
  onUploadProgress: ((progressEvent: AxiosProgressEvent) => void) | undefined,
  totalBytes: number,
  loadedBytes: number,
): void {
  if (!onUploadProgress) {
    return;
  }

  const total = Math.max(totalBytes, 0);
  const loaded = Math.max(0, Math.min(loadedBytes, total));

  onUploadProgress({
    loaded,
    total,
    progress: total > 0 ? loaded / total : undefined,
    lengthComputable: total > 0,
    upload: true,
  } as AxiosProgressEvent);
}

function isPayloadTooLargeError(error: unknown): boolean {
  return axios.isAxiosError(error) && error.response?.status === 413;
}

export async function getKnowledgeObject(
  id: string,
): Promise<KnowledgeObjectDetails> {
  const response = await api.get<BackendKnowledgeObjectDetails>(
    `/api/v1/data/${id}`,
  );
  return mapKnowledgeObjectDetails(response.data);
}

export async function reprocessKnowledgeObject(
  id: string,
): Promise<KnowledgeObject> {
  const response = await api.post<BackendKnowledgeObject>(
    `/api/v1/data/${id}/reprocess`,
  );
  return mapKnowledgeObject(response.data);
}

export async function deleteKnowledgeObject(id: string): Promise<void> {
  await api.delete(`/api/v1/data/${id}`);
}

export async function downloadKnowledgeObject(
  id: string,
): Promise<{ blob: Blob; filename?: string }> {
  const response = await api.get<Blob>(`/api/v1/data/${id}/download`, {
    responseType: "blob",
  });

  return {
    blob: response.data,
    filename: parseFilenameFromDisposition(
      response.headers["content-disposition"],
    ),
  };
}

function parseFilenameFromDisposition(header?: string): string | undefined {
  if (!header) {
    return undefined;
  }

  const match = /filename\*=UTF-8''([^;]+)|filename="?([^";]+)"?/i.exec(header);
  const rawFilename = match?.[1] || match?.[2];
  if (!rawFilename) {
    return undefined;
  }

  try {
    return decodeURIComponent(rawFilename);
  } catch {
    return rawFilename;
  }
}

export function isAxiosError(
  error: unknown,
): error is ReturnType<typeof axios.isAxiosError> {
  return axios.isAxiosError(error);
}

type BackendKnowledgeObjectDetails = BackendKnowledgeObject & {
  content?: string;
  chunks?: KnowledgeObjectDetails["chunks"];
};

function mapKnowledgeObject(item: BackendKnowledgeObject): KnowledgeObject {
  return {
    id: item.id,
    filename: item.filename,
    originalFilename: item.filename,
    type: item.type,
    mimeType: item.contentType,
    contentType: item.contentType,
    size: item.sizeBytes,
    sizeBytes: item.sizeBytes,
    status: normalizeKnowledgeObjectStatus(item.status),
    errorMessage: item.errorMessage,
    createdAt: item.createdAt,
    updatedAt: item.updatedAt,
    metadata: item.metadata,
    tags: item.tags ?? [],
    title: item.title,
    sha256: item.sha256,
    hasContent: item.hasContent,
    hasResult: item.hasResult,
    outputFormat: item.outputFormat,
    language: item.language,
  };
}

function mapKnowledgeObjectDetails(
  item: BackendKnowledgeObjectDetails,
): KnowledgeObjectDetails {
  return {
    ...mapKnowledgeObject(item),
    content: item.content,
    chunks: item.chunks,
  };
}

function normalizeKnowledgeObjectStatus(
  status?: string,
): KnowledgeObjectStatus {
  switch (status) {
    case "completed":
    case "ready":
      return "ready";
    case "processing":
      return "processing";
    case "failed":
    case "error":
      return "failed";
    case "pending":
    default:
      return "pending";
  }
}
