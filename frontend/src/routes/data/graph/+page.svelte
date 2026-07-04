<script lang="ts">
    import {Button} from "$lib/components/ui/button/index.js";
    import {Input} from "$lib/components/ui/input/index.js";
    import GraphViewer from "$lib/components/graph/GraphViewer.svelte";
    import {type GraphData, type GraphNode, type GraphNodeType, NODE_COLORS} from "$lib/data/graph";
    import {queryKnowledgeGraph} from "$lib/api/data";
    import {XIcon} from "@lucide/svelte";
    import {getApiErrorMessage} from "$lib/api/auth";

    let query = $state("");
    let isLoading = $state(false);
    let errorMessage = $state("");
    let graphData = $state<GraphData>({nodes: [], edges: []});

    let selectedNode = $state<GraphNode | null>(null);

    const nodeTypeEntries = $derived(Object.entries(NODE_COLORS) as [GraphNodeType, string][]);

    function closeDetails() {
        selectedNode = null;
    }

    async function handleSubmit(event: SubmitEvent) {
        event.preventDefault();
        const trimmed = query.trim();
        if (!trimmed || isLoading) {
            return;
        }

        isLoading = true;
        errorMessage = "";

        try {
            graphData = await queryKnowledgeGraph(trimmed);
        } catch (error) {
            errorMessage = getApiErrorMessage(error, "Не удалось загрузить граф знаний.");
        } finally {
            isLoading = false;
        }
    }
</script>

<svelte:head>
    <title>Карта знаний</title>
</svelte:head>

<main class="relative flex min-h-0 flex-1 flex-col overflow-hidden">
    <div class="z-10 flex flex-col gap-4 px-6 py-4 md:flex-row md:items-center md:justify-between w-full">
        <form class="flex gap-2 w-full" onsubmit={handleSubmit}>
            <div class="relative w-full max-w-lg">
                <Input
                    bind:value={query}
                    placeholder="Например: никель, электроэкстракция"
                    class="h-10 pl-3 w-full"
                    disabled={isLoading}
                />
            </div>
            <Button type="submit" variant="outline" disabled={isLoading || !query.trim()}>
                {isLoading ? "Загрузка..." : "Построить"}
            </Button>
        </form>
    </div>

    <div class="relative flex min-h-0 flex-1">
        {#if errorMessage}
            <div class="absolute inset-0 z-20 flex items-start justify-center pt-4 px-4">
                <div class="text-sm text-destructive">
                    {errorMessage}
                </div>
            </div>
        {/if}

        {#key graphData}
            <GraphViewer data={graphData} onNodeSelect={(node) => (selectedNode = node)} />
        {/key}

        <div class="pointer-events-none absolute right-4 bottom-4 z-10 w-[calc(100%-2rem)] max-w-sm rounded-2xl border border-border/20 bg-muted/20 p-4 sm:w-80">
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
            <div class="absolute inset-0 z-30 flex items-center justify-center bg-black/40 p-4" role="presentation" onclick={closeDetails}>
                <div
                    class="relative w-full max-w-md rounded-2xl border bg-background p-6"
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
    </div>
</main>
