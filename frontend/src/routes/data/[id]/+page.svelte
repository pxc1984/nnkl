<script lang="ts">
	import { browser } from "$app/environment";
	import { page } from "$app/state";
	import { onMount } from "svelte";
	import { getApiErrorMessage } from "$lib/api/auth";
	import { downloadKnowledgeObject, getKnowledgeObject, reprocessKnowledgeObject } from "$lib/api/data";
	import DataPageHeader from "$lib/components/data/data-page-header.svelte";
	import DataStatusBadge from "$lib/components/data/data-status-badge.svelte";
	import { Button } from "$lib/components/ui/button/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import { Skeleton } from "$lib/components/ui/skeleton/index.js";
	import { Badge } from "$lib/components/ui/badge/index.js";
	import type { KnowledgeObjectDetails } from "$lib/data/types";
	import {
		formatBytes,
		formatDateTime,
		getContentPreview,
		getMetadataEntries,
		getObjectTitle,
		getObjectTypeLabel,
	} from "$lib/data/utils";
	import DownloadIcon from "@lucide/svelte/icons/download";
	import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";

	const LOADING_SKELETON_DELAY_MS = 150;

	let object = $state<KnowledgeObjectDetails | null>(null);
	let isLoading = $state(false);
	let isDownloading = $state(false);
	let isReprocessing = $state(false);
	let errorMessage = $state("");
	let showFullContent = $state(false);
	let isMounted = $state(false);

	onMount(() => {
		isMounted = true;

		return () => {
			isMounted = false;
		};
	});

	$effect(() => {
		const id = page.params.id;
		if (!browser || !isMounted || !id) {
			return;
		}

		object = null;
		isLoading = false;
		errorMessage = "";
		let cancelled = false;
		const loadingTimer = window.setTimeout(() => {
			if (!cancelled) {
				isLoading = true;
			}
		}, LOADING_SKELETON_DELAY_MS);

		getKnowledgeObject(id)
			.then((response) => {
				if (cancelled) return;
				object = response;
				showFullContent = false;
			})
			.catch((error) => {
				if (cancelled) return;
				errorMessage = getApiErrorMessage(error, "Не удалось загрузить документ.");
				object = null;
			})
			.finally(() => {
				window.clearTimeout(loadingTimer);
				if (!cancelled) {
					isLoading = false;
				}
			});

		return () => {
			cancelled = true;
			window.clearTimeout(loadingTimer);
		};
	});

	async function handleDownload(): Promise<void> {
		if (!object || isDownloading) {
			return;
		}

		isDownloading = true;

		try {
			const { blob, filename } = await downloadKnowledgeObject(object.id);
			const objectUrl = URL.createObjectURL(blob);
			const anchor = document.createElement("a");
			anchor.href = objectUrl;
			anchor.download = filename || object.originalFilename || object.filename;
			document.body.append(anchor);
			anchor.click();
			anchor.remove();
			URL.revokeObjectURL(objectUrl);
		} catch (error) {
			errorMessage = getApiErrorMessage(error, "Не удалось скачать документ.");
		} finally {
			isDownloading = false;
		}
	}

	async function handleReprocess(): Promise<void> {
		if (!object || isReprocessing) {
			return;
		}

		isReprocessing = true;
		errorMessage = "";

		try {
			const updatedObject = await reprocessKnowledgeObject(object.id);
			object = { ...object, ...updatedObject };
		} catch (error) {
			errorMessage = getApiErrorMessage(error, "Не удалось перезапустить обработку.");
		} finally {
			isReprocessing = false;
		}
	}

	const contentState = $derived(getContentPreview(object?.content));
	const metadataEntries = $derived(getMetadataEntries(object?.metadata));
</script>

<div class="flex flex-col gap-8">
	<DataPageHeader
		title={object ? getObjectTitle(object) : "Документ"}
	>
		{#snippet actions()}
			<Button variant="outline" class="rounded-full" disabled={!object || isReprocessing} onclick={() => void handleReprocess()}>
				<RefreshCwIcon class="size-4" />
				{isReprocessing ? "Перезапускаем..." : "Переобработать"}
			</Button>
			<Button class="rounded-full" disabled={!object || isDownloading} onclick={() => void handleDownload()}>
				<DownloadIcon class="size-4" />
				{isDownloading ? "Скачиваем..." : "Скачать"}
			</Button>
		{/snippet}
	</DataPageHeader>

	{#if errorMessage}
		<div class="text-destructive bg-destructive/10 rounded-2xl border border-destructive/20 px-4 py-3 text-sm">{errorMessage}</div>
	{/if}

	{#if isLoading && !object}
		<div class="grid gap-6 xl:grid-cols-[minmax(0,1.8fr)_minmax(20rem,0.9fr)]">
			<div class="space-y-6">
				<div class="bg-card/90 rounded-[1.75rem] border border-border/60 p-6"><Skeleton class="h-7 w-64 rounded-full" /></div>
				<div class="bg-card/90 rounded-[1.75rem] border border-border/60 p-6"><Skeleton class="h-72 w-full rounded-2xl" /></div>
			</div>
			<div class="bg-card/90 rounded-[1.75rem] border border-border/60 p-6"><Skeleton class="h-80 w-full rounded-2xl" /></div>
		</div>
	{:else if object}
		<div class="grid gap-6 xl:grid-cols-[minmax(0,1.8fr)_minmax(20rem,0.9fr)]">
			<div class="space-y-6">
				<Card.Root class="bg-card/90 rounded-[1.75rem] border-border/60 shadow-[0_20px_60px_-36px_rgba(0,0,0,0.35)]">
					<Card.Header class="gap-4 border-b border-border/60 pb-5">
						<div class="flex flex-wrap items-center gap-2">
							<Card.Title class="text-lg">Сводка</Card.Title>
							<DataStatusBadge status={object.status} />
						</div>
						<Card.Description>Основная информация по загруженному объекту и его текущему состоянию.</Card.Description>
					</Card.Header>
					<Card.Content class="grid gap-4 pt-5 md:grid-cols-2">
						<div>
							<p class="text-muted-foreground text-sm">Тип</p>
							<p class="mt-1 text-sm font-medium">{getObjectTypeLabel(object)}</p>
						</div>
						<div>
							<p class="text-muted-foreground text-sm">Размер</p>
							<p class="mt-1 text-sm font-medium">{formatBytes(object.size)}</p>
						</div>
						<div>
							<p class="text-muted-foreground text-sm">Загружен</p>
							<p class="mt-1 text-sm font-medium">{formatDateTime(object.createdAt)}</p>
						</div>
						<div>
							<p class="text-muted-foreground text-sm">Обновлен</p>
							<p class="mt-1 text-sm font-medium">{formatDateTime(object.updatedAt)}</p>
						</div>
						<div class="md:col-span-2">
							<p class="text-muted-foreground text-sm">Автор</p>
							<p class="mt-1 text-sm font-medium">{object.createdBy?.name || object.createdBy?.email || "-"}</p>
						</div>
						{#if object.errorMessage}
							<div class="text-destructive bg-destructive/10 md:col-span-2 rounded-2xl border border-destructive/20 px-4 py-3 text-sm">
								{object.errorMessage}
							</div>
						{/if}
					</Card.Content>
				</Card.Root>

				<Card.Root class="bg-card/90 rounded-[1.75rem] border-border/60 shadow-[0_20px_60px_-36px_rgba(0,0,0,0.35)]">
					<Card.Header class="gap-2 border-b border-border/60 pb-5">
						<Card.Title class="text-lg">Содержимое</Card.Title>
						<Card.Description>
							{#if object.status === "ready"}
								Текст, который был извлечен из документа и доступен для поиска.
							{:else}
								Содержимое появится после завершения обработки.
							{/if}
						</Card.Description>
					</Card.Header>
					<Card.Content class="space-y-4 pt-5">
						{#if object.content}
							<pre class="bg-background/80 max-h-[32rem] overflow-auto rounded-2xl border border-border/60 p-4 text-sm leading-6 whitespace-pre-wrap break-words">{showFullContent ? object.content : contentState.text}</pre>
							{#if contentState.truncated}
								<Button type="button" variant="ghost" class="rounded-full" onclick={() => (showFullContent = !showFullContent)}>
									{showFullContent ? "Свернуть" : "Показать полностью"}
								</Button>
							{/if}
						{:else}
							<div class="bg-background/70 text-muted-foreground rounded-2xl border border-border/60 px-4 py-6 text-sm">
								{object.status === "failed"
									? "Не удалось извлечь текст из документа."
									: "Текст документа пока недоступен. Вероятно, объект еще обрабатывается."}
							</div>
						{/if}

						{#if object.chunks?.length}
							<div class="rounded-2xl border border-border/60 px-4 py-4">
								<div class="flex items-center justify-between gap-3">
									<div>
										<p class="text-sm font-medium">Чанки</p>
										<p class="text-muted-foreground text-sm">Подготовлено {object.chunks.length} фрагментов для поиска.</p>
									</div>
								</div>
								<div class="mt-4 space-y-3">
									{#each object.chunks.slice(0, 3) as chunk (`${chunk.index}:${chunk.text.slice(0, 20)}`)}
										<div class="bg-background/80 rounded-2xl border border-border/60 px-4 py-3">
											<div class="mb-2 flex items-center justify-between gap-3 text-sm">
												<span class="font-medium">Фрагмент {chunk.index + 1}</span>
												<span class="text-muted-foreground">{chunk.tokens ?? 0} токенов</span>
											</div>
											<p class="text-sm leading-6 whitespace-pre-wrap break-words">{chunk.text}</p>
										</div>
									{/each}
								</div>
							</div>
						{/if}
					</Card.Content>
				</Card.Root>
			</div>

			<div class="space-y-6">
				<Card.Root class="bg-card/90 rounded-[1.75rem] border-border/60 shadow-[0_20px_60px_-36px_rgba(0,0,0,0.35)]">
					<Card.Header class="gap-2 border-b border-border/60 pb-5">
						<Card.Title class="text-lg">Метки и метаданные</Card.Title>
						<Card.Description>Дополнительные данные, полезные для навигации и фильтрации.</Card.Description>
					</Card.Header>
					<Card.Content class="space-y-5 pt-5">
						<div>
							<p class="text-muted-foreground mb-2 text-sm">Теги</p>
							{#if object.tags?.length}
								<div class="flex flex-wrap gap-2">
									{#each object.tags as tag (tag)}
										<Badge variant="outline">{tag}</Badge>
									{/each}
								</div>
							{:else}
								<p class="text-sm">-</p>
							{/if}
						</div>

						<div>
							<p class="text-muted-foreground mb-3 text-sm">Метаданные</p>
							{#if metadataEntries.length > 0}
								<div class="space-y-3">
									{#each metadataEntries as [key, value] (`${key}:${value}`)}
										<div class="rounded-2xl border border-border/60 px-4 py-3">
											<p class="text-muted-foreground text-sm">{key}</p>
											<p class="mt-1 text-sm leading-6 break-words">{value}</p>
										</div>
									{/each}
								</div>
							{:else}
								<p class="text-sm">Метаданные не переданы.</p>
							{/if}
						</div>
					</Card.Content>
				</Card.Root>
			</div>
		</div>
	{/if}
</div>
