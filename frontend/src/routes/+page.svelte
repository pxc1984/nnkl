<script lang="ts">
    import AppSidebar from "$lib/components/app-sidebar.svelte";
    import { Button } from "$lib/components/ui/button/index.js";
    import * as Sidebar from "$lib/components/ui/sidebar/index.js";
    import { ArrowUpIcon, GlobeIcon } from "@lucide/svelte";

    let prompt = "";
    let useDomesticSources = false;
</script>

<Sidebar.Provider>
    <AppSidebar />
    <Sidebar.Inset class="bg-background min-h-screen">
        <header class="flex h-16 shrink-0 items-center justify-between px-4 md:px-6">
            <div class="flex items-center gap-3">
                <Sidebar.Trigger class="-ms-1" />
            </div>
        </header>
        <main class="flex flex-1 items-center justify-center px-4 pb-10 pt-4 md:px-8">
            <section class="flex w-full max-w-4xl flex-col items-center gap-8">
                <div class="space-y-3 text-center">
                    <p class="text-muted-foreground text-sm tracking-[0.24em] uppercase">Поиск по базе знаний</p>
                    <h1 class="text-foreground text-3xl font-semibold tracking-tight md:text-5xl">
                        Что у вас сегодня на уме?
                    </h1>
                </div>

                <div class="bg-card/90 w-full rounded-[2rem] border border-border/60 shadow-[0_24px_80px_-32px_rgba(0,0,0,0.45)] backdrop-blur">
                    <div class="border-border/60 bg-background/70 flex min-h-36 flex-col rounded-[1.5rem] border px-4 py-4 md:px-5">
                        <div class="mb-6">
                            <textarea
                                bind:value={prompt}
                                rows="4"
                                placeholder="Какие способы закачки шахтных вод в глубокие горизонты применялись в России и за рубежом, и каковы их технико-экономические показатели?"
                                class="text-foreground placeholder:text-muted-foreground field-sizing-content min-h-28 w-full resize-none border-0 bg-transparent px-0 py-0 text-base leading-7 shadow-none outline-none focus-visible:border-0 focus-visible:ring-0 md:text-lg"
                            ></textarea>
                        </div>

                        <div class="mt-auto flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
                            <div class="flex flex-wrap gap-2">
                                <button
                                    type="button"
                                    class={useDomesticSources
                                        ? "bg-blue-600 text-white hover:bg-blue-500 inline-flex items-center gap-2 rounded-full px-3 py-2 text-sm transition-colors"
                                        : "bg-muted text-muted-foreground hover:bg-muted/80 inline-flex items-center gap-2 rounded-full px-3 py-2 text-sm transition-colors"}
                                    on:click={() => (useDomesticSources = !useDomesticSources)}
                                >
                                    <GlobeIcon class="size-4" />
                                    {useDomesticSources ? "Отечественные источники" : "Все источники"}
                                </button>
                            </div>

                            <div class="flex items-center justify-end gap-2">
                                <Button type="button" size="icon" class="rounded-full">
                                    <ArrowUpIcon class="size-4" />
                                </Button>
                            </div>
                        </div>
                    </div>
                </div>
            </section>
        </main>
    </Sidebar.Inset>
</Sidebar.Provider>
