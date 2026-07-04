<script lang="ts">
  import AppSidebar from "$lib/components/app-sidebar.svelte";
  import { Button } from "$lib/components/ui/button/index.js";
  import * as Sidebar from "$lib/components/ui/sidebar/index.js";
  import { authState } from "$lib/auth/store";
  import GraphViewer from "$lib/components/graph/GraphViewer.svelte";
  import { mockGraph, NODE_COLORS, type GraphNode, type GraphNodeType } from "$lib/data/graph";
  import { RotateCcwIcon, InfoIcon } from "@lucide/svelte";

  let selectedNode = $state<GraphNode | null>(null);

  function handleReset() {
    selectedNode = null;
  }

  const nodeTypeEntries = $derived(Object.entries(NODE_COLORS) as [GraphNodeType, string][]);
</script>

<Sidebar.Provider>
  <AppSidebar currentUser={$authState.user} />
  <Sidebar.Inset class="bg-background flex min-h-screen flex-col">
    <header class="flex h-16 shrink-0 items-center justify-between border-b px-4 md:px-6">
      <div class="flex items-center gap-3">
        <Sidebar.Trigger class="-ms-1" />
        <h1 class="text-lg font-semibold">Карта знаний</h1>
      </div>
      <Button variant="outline" size="sm" onclick={handleReset}>
        <RotateCcwIcon class="mr-2 size-4" />
        Сбросить выбор
      </Button>
    </header>

    <main class="flex flex-1 gap-4 overflow-hidden p-4">
      <section class="flex flex-1 flex-col overflow-hidden rounded-xl border bg-card shadow-sm">
        <div class="border-b px-4 py-3">
          <p class="text-sm text-muted-foreground">
            Визуализация связей между материалами, процессами, оборудованием и экспертами
          </p>
        </div>
        <div class="min-h-0 flex-1 p-2">
          <GraphViewer data={mockGraph} onNodeSelect={(node) => (selectedNode = node)} />
        </div>
      </section>

      <aside class="flex w-80 shrink-0 flex-col overflow-hidden rounded-xl border bg-card shadow-sm">
        <div class="border-b px-4 py-3">
          <h2 class="font-semibold">Детали</h2>
        </div>
        <div class="flex-1 overflow-auto p-4">
          {#if selectedNode}
            <div class="space-y-4">
              <div>
                <p class="text-xs uppercase tracking-wider text-muted-foreground">Сущность</p>
                <p class="text-lg font-medium">{selectedNode.label}</p>
              </div>
              <div>
                <p class="text-xs uppercase tracking-wider text-muted-foreground">Тип</p>
                <div class="mt-1 flex items-center gap-2">
                  <span
                    class="inline-block size-3 rounded-full"
                    style="background-color: {NODE_COLORS[selectedNode.type]}"
                  ></span>
                  <span class="text-sm">{selectedNode.type}</span>
                </div>
              </div>
              <div>
                <p class="text-xs uppercase tracking-wider text-muted-foreground">ID</p>
                <p class="break-all font-mono text-sm text-muted-foreground">{selectedNode.id}</p>
              </div>
            </div>
          {:else}
            <div class="flex h-full flex-col items-center justify-center gap-2 text-center text-muted-foreground">
              <InfoIcon class="size-8 opacity-40" />
              <p class="text-sm">Кликните на узел графа, чтобы увидеть детали</p>
            </div>
          {/if}
        </div>

        <div class="border-t p-4">
          <p class="mb-2 text-xs font-medium text-muted-foreground">Легенда</p>
          <div class="grid grid-cols-2 gap-2">
            {#each nodeTypeEntries as [type, color] (type)}
              <div class="flex items-center gap-2">
                <span class="inline-block size-2.5 rounded-full" style="background-color: {color}"></span>
                <span class="text-xs text-muted-foreground">{type}</span>
              </div>
            {/each}
          </div>
        </div>
      </aside>
    </main>
  </Sidebar.Inset>
</Sidebar.Provider>
