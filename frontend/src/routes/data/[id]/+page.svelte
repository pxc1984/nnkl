<script lang="ts">
    import {browser} from "$app/environment";
    import {resolve} from "$app/paths";
    import {page} from "$app/state";
    import {getApiErrorMessage} from "$lib/api/auth";
    import {downloadKnowledgeObject, getKnowledgeObject, reprocessKnowledgeObject,} from "$lib/api/data";
    import DataStatusBadge from "$lib/components/data/data-status-badge.svelte";
    import MarkdownRenderer from "$lib/components/markdown-renderer.svelte";
    import {Badge} from "$lib/components/ui/badge/index.js";
    import {Button} from "$lib/components/ui/button/index.js";
    import {
        formatBytes,
        formatDateTime,
        getMetadataEntries,
        getObjectTitle,
        getObjectTypeLabel,
    } from "$lib/data/utils";
    import type {KnowledgeObjectDetails} from "$lib/data/types";
    import { ChevronLeftIcon, DownloadIcon, FileSearchIcon, LoaderIcon } from "@lucide/svelte";

    let object = $state<KnowledgeObjectDetails | null>(null);
    let isLoading = $state(false);
    let isReprocessing = $state(false);
    let isDownloading = $state(false);
    let errorMessage = $state("");
    let requestRun = 0;

    const objectId = $derived(page.params.id);
    const metadataEntries = $derived(getMetadataEntries(object?.metadata));
    const hasMarkdownContent = $derived(Boolean(object?.content?.trim()));

    async function loadObject(): Promise<void> {
        if (!objectId) {
            object = null;
            errorMessage = "Не указан идентификатор документа.";
            return;
        }

        const currentRun = ++requestRun;
        isLoading = true;
        errorMessage = "";

        try {
            const response = await getKnowledgeObject(objectId);
            if (currentRun !== requestRun) {
                return;
            }

            object = response;
        } catch (error) {
            if (currentRun !== requestRun) {
                return;
            }

            object = null;
            errorMessage = getApiErrorMessage(error, "Не удалось загрузить документ.");
        } finally {
            if (currentRun === requestRun) {
                isLoading = false;
            }
        }
    }

    async function handleReprocess(): Promise<void> {
        if (!objectId || isReprocessing) {
            return;
        }

        isReprocessing = true;
        errorMessage = "";
        try {
            const updated = await reprocessKnowledgeObject(objectId);
            object = object ? {...object, ...updated} : {...updated};
            await loadObject();
        } catch (error) {
            errorMessage = getApiErrorMessage(error, "Не удалось запустить повторную OCR-обработку.");
        } finally {
            isReprocessing = false;
        }
    }

    async function handleDownload(): Promise<void> {
        if (!objectId || isDownloading || !browser) {
            return;
        }

        isDownloading = true;
        errorMessage = "";
        try {
            const {blob, filename} = await downloadKnowledgeObject(objectId);
            const url = URL.createObjectURL(blob);
            const link = document.createElement("a");
            link.href = url;
            link.download = filename || object?.originalFilename || object?.filename || `${objectId}.bin`;
            document.body.append(link);
            link.click();
            link.remove();
            URL.revokeObjectURL(url);
        } catch (error) {
            errorMessage = getApiErrorMessage(error, "Не удалось скачать файл.");
        } finally {
            isDownloading = false;
        }
    }

    $effect(() => {
        if (!browser) {
            return;
        }

        void loadObject();
    });
</script>

<div class="mx-auto flex w-full max-w-6xl flex-col py-4">

    <div class="mb-10 flex items-center justify-between">
        <Button href={resolve("/data")} variant="ghost">
            <ChevronLeftIcon class="size-4"/>
            К списку
        </Button>

        <div class="flex gap-2">
            <Button
                    variant="outline"
                    disabled={isReprocessing || isLoading || !object}
                    onclick={() => void handleReprocess()}
            >
                {#if isReprocessing}
                    <LoaderIcon class="size-4 animate-spin"/>
                {:else}
                    <FileSearchIcon class="size-4"/>
                {/if}

                Пере-OCR
            </Button>

            <Button
                    disabled={isDownloading || isLoading || !object}
                    onclick={() => void handleDownload()}
            >
                {#if isDownloading}
                    <LoaderIcon class="size-4 animate-spin"/>
                {:else}
                    <DownloadIcon class="size-4"/>
                {/if}

                Скачать
            </Button>
        </div>
    </div>

    {#if errorMessage && !object}
        <div class="rounded-2xl border border-destructive/20 bg-destructive/10 px-4 py-3 text-sm text-destructive">
            {errorMessage}
        </div>
    {:else if object}
        <div class="grid gap-14 lg:grid-cols-[minmax(0,1fr)_18rem]">

            <div class="space-y-14">

                <section>
                    <div class="mb-6 flex flex-wrap items-center gap-3">
                        <h1 class="text-3xl font-semibold tracking-tight">
                            {getObjectTitle(object)}
                        </h1>

                        <DataStatusBadge status={object.status}/>
                        <Badge variant="outline">{getObjectTypeLabel(object)}</Badge>
                    </div>

                    {#if object.errorMessage}
                        <p class="mb-6 text-sm text-destructive">
                            {object.errorMessage}
                        </p>
                    {/if}

                    {#if object.tags?.length}
                        <div class="mb-8 flex flex-wrap gap-2">
                            {#each object.tags as tag (tag)}
                                <Badge variant="secondary">{tag}</Badge>
                            {/each}
                        </div>
                    {/if}

                    <div class="grid gap-6 sm:grid-cols-3 text-sm">
                        <div>
                            <div class="text-muted-foreground">
                                Размер
                            </div>

                            <div>{formatBytes(object.size)}</div>
                        </div>

                        <div>
                            <div class="text-muted-foreground">
                                Создан
                            </div>

                            <div>{formatDateTime(object.createdAt)}</div>
                        </div>

                        <div>
                            <div class="text-muted-foreground">
                                Обновлён
                            </div>

                            <div>{formatDateTime(object.updatedAt)}</div>
                        </div>
                    </div>
                </section>

                <section>
                    <div class="mb-8 flex items-center justify-between">
                        <h2 class="text-xl font-medium">
                            Markdown
                        </h2>

                        {#if object.outputFormat}
                            <div class="text-sm text-muted-foreground">
                                {object.outputFormat}
                            </div>
                        {/if}
                    </div>

                    {#if hasMarkdownContent}
                        <MarkdownRenderer markdown={object.content ?? ""}/>
                    {:else}
                        <p class="text-muted-foreground">
                            Отпаршенный markdown пока недоступен.
                        </p>
                    {/if}
                </section>

                {#if metadataEntries.length}
                    <section>
                        <h2 class="mb-6 text-xl font-medium">
                            Дополнительные метаданные
                        </h2>

                        <div class="grid gap-5 sm:grid-cols-2">
                            {#each metadataEntries as [key, value] (key)}
                                <div>
                                    <div class="text-xs uppercase text-muted-foreground">
                                        {key}
                                    </div>

                                    <div class="mt-1 break-words">
                                        {value}
                                    </div>
                                </div>
                            {/each}
                        </div>
                    </section>
                {/if}

            </div>

            <aside class="h-fit rounded-2xl border border-border/20 bg-muted/20 p-5">
                <h2 class="mb-6 font-medium">
                    Системные поля
                </h2>

                <dl class="space-y-5 text-sm">

                    <div>
                        <dt class="text-muted-foreground">ID</dt>
                        <dd class="mt-1 font-mono text-xs break-all">{object.id}</dd>
                    </div>

                    <div>
                        <dt class="text-muted-foreground">Имя файла</dt>
                        <dd class="mt-1">{object.originalFilename || object.filename}</dd>
                    </div>

                    <div>
                        <dt class="text-muted-foreground">Тип</dt>
                        <dd class="mt-1">{object.type || "-"}</dd>
                    </div>

                    <div>
                        <dt class="text-muted-foreground">Content-Type</dt>
                        <dd class="mt-1 break-all">{object.contentType || object.mimeType || "-"}</dd>
                    </div>

                    <div>
                        <dt class="text-muted-foreground">SHA-256</dt>
                        <dd class="mt-1 font-mono text-xs break-all">{object.sha256 || "-"}</dd>
                    </div>

                </dl>
            </aside>

        </div>
    {/if}

</div>
