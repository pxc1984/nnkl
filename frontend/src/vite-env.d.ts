/// <reference types="@sveltejs/kit" />

interface ImportMetaEnv {
  readonly API_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
