# frontend-svelte-shadcn

Svelte 5 + TypeScript + Vite starter using `pnpm`, Tailwind CSS v4, and real `shadcn-svelte` components.

## Requirements

- Node.js 20+
- `pnpm`

## Getting Started

Install dependencies:

```bash
pnpm install
```

Optional: override the backend API URL:

```bash
cp .env.example .env
```

Start the dev server:

```bash
pnpm run dev
```

Open the local URL printed by Vite.

## Available Scripts

Run the development server:

```bash
pnpm run dev
```

Build for production:

```bash
pnpm run build
```

Preview the production build locally:

```bash
pnpm run preview
```

Run Svelte and TypeScript checks:

```bash
pnpm run check
```

Run ESLint:

```bash
pnpm run lint
```

## Project Structure

```text
src/
  App.svelte                 Main starter screen
  app.css                    Global Tailwind/theme styles
  main.ts                    App bootstrap, enables dark mode by default
  lib/
    api/
      client.ts              Shared Axios instance
      health.ts              Example health endpoint call
    config.ts                Frontend config values
    utils.ts                 `cn` helper and shared types
    components/ui/           Generated shadcn-svelte UI components
  vite-env.d.ts              Typed Vite env variables
components.json              shadcn-svelte CLI configuration
vite.config.ts               Vite config with Tailwind and $lib alias
tsconfig*.json               TypeScript config
eslint.config.js             ESLint config
.env.example                 Example environment variables
```

## Styling

- Tailwind CSS v4 is imported in `src/app.css`
- Geist Variable font is loaded globally
- Theme tokens are defined with CSS variables in `src/app.css`
- Dark mode is enabled by default in `src/main.ts`

## Using shadcn-svelte

This project is configured for the `shadcn-svelte` CLI through `components.json`.

Add a new component:

```bash
pnpm dlx shadcn-svelte@latest add <component-name>
```

Example:

```bash
pnpm dlx shadcn-svelte@latest add dialog
```

Overwrite an existing generated component:

```bash
pnpm dlx shadcn-svelte@latest add <component-name> -o
```

## Import Conventions

Import generated UI components from their package-style entrypoints:

```ts
import { Button } from "$lib/components/ui/button";
import * as Card from "$lib/components/ui/card";
import * as Avatar from "$lib/components/ui/avatar";
```

Use `$lib` for shared code under `src/lib`.

## API Setup

This template includes Axios and a small API layer.

Environment variable:

```bash
API_URL=
```

Behavior:

- uses `API_URL` when set
- uses same-origin requests when `API_URL` is empty or unset
- Vite proxies `/api` to `http://localhost:8080` during `pnpm run dev`

Axios client:

```ts
import { api } from "$lib/api/client";
```

Example endpoint helper:

```ts
import { getHealth } from "$lib/api/health";

const health = await getHealth();
```

That helper calls:

```text
GET /api/health
```

The starter screen already shows a live example request using this setup.

## Typical Workflow

1. Install dependencies with `pnpm install`.
2. Start development with `pnpm run dev`.
3. Edit `src/App.svelte` or add routes/components for your app.
4. Add more `shadcn-svelte` components with the CLI as needed.
5. Run `pnpm run check` and `pnpm run lint` before finishing changes.
6. Run `pnpm run build` to verify the production bundle.

## Notes

- This template is a client-side Svelte SPA built with Vite.
- The initial screen is starter content and is meant to be replaced.
- If you change aliases, update both `tsconfig.json` and `vite.config.ts`.
