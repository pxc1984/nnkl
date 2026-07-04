import axios, { type AxiosProgressEvent } from "axios";

import { api } from "$lib/api/client";
import { DEFAULT_DATA_PAGE_SIZE } from "$lib/data/utils";
import type {
  DataListParams,
  DataTagList,
  DataUploadParams,
  DataUploadResponse,
  KnowledgeObject,
  KnowledgeObjectDetails,
  PaginatedKnowledgeObjectList,
} from "$lib/data/types";

export async function listKnowledgeObjects(
  params: DataListParams,
): Promise<PaginatedKnowledgeObjectList> {
  const response = await api.get<PaginatedKnowledgeObjectList>("/api/v1/data", {
    params: {
      page: params.page,
      pageSize: params.pageSize,
      query: params.query,
      type: params.type,
      tags: params.tags,
    },
    validateStatus: (status) => status === 200 || status === 204,
  });

  if (response.status === 204) {
    return {
      items: [],
      meta: {
        page: params.page ?? 1,
        pageSize: params.pageSize ?? DEFAULT_DATA_PAGE_SIZE,
        total: 0,
        totalPages: 0,
      },
    };
  }

  return response.data;
}

export async function listDataTags(): Promise<DataTagList> {
  const response = await api.get<DataTagList>("/api/v1/tags");
  return response.data;
}

export async function uploadKnowledgeObjects(
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
      headers: {
        "Content-Type": "multipart/form-data",
      },
      onUploadProgress,
    },
  );

  return response.data;
}

export async function getKnowledgeObject(
  id: string,
): Promise<KnowledgeObjectDetails> {
  const response = await api.get<KnowledgeObjectDetails>(`/api/v1/data/${id}`);
  return response.data;
}

export async function reprocessKnowledgeObject(
  id: string,
): Promise<KnowledgeObject> {
  const response = await api.post<KnowledgeObject>(
    `/api/v1/data/${id}/reprocess`,
  );
  return response.data;
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
