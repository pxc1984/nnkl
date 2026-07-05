export type KnowledgeObjectStatus =
  "pending" | "processing" | "ready" | "failed";

export type UserShort = {
  id: string;
  email: string;
  name?: string;
};

export type KnowledgeChunk = {
  index: number;
  text: string;
  tokens?: number;
};

export type KnowledgeObject = {
  id: string;
  filename: string;
  originalFilename?: string;
  type?: string;
  mimeType?: string;
  contentType?: string;
  size?: number;
  sizeBytes?: number;
  status: KnowledgeObjectStatus;
  errorMessage?: string | null;
  createdAt: string;
  updatedAt?: string;
  createdBy?: UserShort;
  metadata?: Record<string, unknown>;
  tags?: string[];
  title?: string;
  sha256?: string;
  hasContent?: boolean;
  hasResult?: boolean;
  outputFormat?: string;
  language?: string | null;
};

export type KnowledgeObjectDetails = KnowledgeObject & {
  content?: string;
  metadata?: Record<string, unknown>;
  chunks?: KnowledgeChunk[];
};

export type PaginationMeta = {
  page: number;
  pageSize: number;
  total: number;
  totalPages?: number;
};

export type PaginatedKnowledgeObjectList = {
  items: KnowledgeObject[];
  meta: PaginationMeta;
};

export type DataTag = {
  name: string;
  count: number;
};

export type DataTagList = {
  items: DataTag[];
};

export type DataListParams = {
  page?: number;
  pageSize?: number;
  query?: string;
  type?: string;
  status?: KnowledgeObjectStatus;
  language?: string;
};

export type DataUploadParams = {
  title?: string;
  tags?: string[];
  recursive?: boolean;
  extractImages?: boolean;
  ocrLanguage?: string;
};

export type DataUploadResponse = {
  items: Array<{
    id: string;
    filename: string;
    type: string;
    status: string;
  }>;
};
