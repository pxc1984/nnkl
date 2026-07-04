<script lang="ts">
    import GraphViewer from "$lib/components/graph/GraphViewer.svelte";
    import {type GraphNode, type GraphNodeType, mockGraph, NODE_COLORS} from "$lib/data/graph";
    import {XIcon} from "@lucide/svelte";

    let selectedNode = $state<GraphNode | null>(null);

    const nodeTypeEntries = $derived(Object.entries(NODE_COLORS) as [GraphNodeType, string][]);

    function closeDetails() {
        selectedNode = null;
    }
</script>

<svelte:head>
    <title>Карта знаний</title>
</svelte:head>

<main class="relative flex min-h-0 flex-1 overflow-hidden rounded-xl">
    <GraphViewer data={mockGraph} onNodeSelect={(node) => (selectedNode = node)}/>

    <div class="pointer-events-none absolute right-4 bottom-4 z-10 w-[calc(100%-2rem)] max-w-sm rounded-xl border bg-card/95 p-4 shadow-lg backdrop-blur sm:w-80">
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

    {#if selectedNode}
        <div class="absolute inset-0 z-20 flex items-center justify-center bg-black/40 p-4 backdrop-blur-sm" role="presentation" onclick={closeDetails}>
            <div
                class="relative w-full max-w-md rounded-2xl border bg-card p-6 shadow-2xl"
                role="dialog"
                tabindex="-1"
                aria-modal="true"
                aria-labelledby="graph-node-title"
                onclick={(event) => event.stopPropagation()}
                onkeydown={(event) => event.stopPropagation()}
            >
                <button
                    type="button"
                    class="absolute top-3 right-3 inline-flex size-8 items-center justify-center rounded-full text-muted-foreground transition hover:bg-muted hover:text-foreground"
                    aria-label="Закрыть"
                    onclick={closeDetails}
                >
                    <XIcon class="size-4"/>
                </button>

                <div class="space-y-4 pr-8">
                    <div>
                        <p class="text-xs uppercase tracking-wider text-muted-foreground">Сущность</p>
                        <p id="graph-node-title" class="text-lg font-medium">{selectedNode.label}</p>
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
            </div>
        </div>
    {/if}
</main>
