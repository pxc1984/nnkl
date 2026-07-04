<script lang="ts">
	import { browser } from "$app/environment";
	import { resolve } from "$app/paths";
	import { page } from "$app/state";
	import { getApiErrorMessage } from "$lib/api/auth";
	import {
		downloadKnowledgeObject,
		getKnowledgeObject,
		reprocessKnowledgeObject,
	} from "$lib/api/data";
	import DataStatusBadge from "$lib/components/data/data-status-badge.svelte";
	import * as Alert from "$lib/components/ui/alert/index.js";
	import { Badge } from "$lib/components/ui/badge/index.js";
	import { Button } from "$lib/components/ui/button/index.js";
	import { Skeleton } from "$lib/components/ui/skeleton/index.js";
	import {
		formatBytes,
		formatDateTime,
		getContentPreview,
		getMetadataEntries,
		getObjectTitle,
		getObjectTypeLabel,
	} from "$lib/data/utils";
	import type { KnowledgeObjectDetails } from "$lib/data/types";
	import ChevronLeftIcon from "@lucide/svelte/icons/chevron-left";
	import DownloadIcon from "@lucide/svelte/icons/download";
	import FileSearchIcon from "@lucide/svelte/icons/file-search";
	import LoaderIcon from "@lucide/svelte/icons/loader";

	let object = $state<KnowledgeObjectDetails | null>(null);
	let isLoading = $state(false);
	let isReprocessing = $state(false);
	let isDownloading = $state(false);
	let errorMessage = $state("");
	let successMessage = $state("");
	let requestRun = 0;

	const objectId = $derived(page.params.id);
	const metadataEntries = $derived(getMetadataEntries(object?.metadata));
	const contentPreview = $derived(getContentPreview(object?.content));

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
		successMessage = "";

		try {
			const updated = await reprocessKnowledgeObject(objectId);
			object = object ? { ...object, ...updated } : { ...updated };
			successMessage = "Повторная OCR-обработка запущена.";
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
		successMessage = "";

		try {
			const { blob, filename } = await downloadKnowledgeObject(objectId);
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

	function getBooleanLabel(value?: boolean): string {
		if (value === true) {
			return "Да";
		}

		if (value === false) {
			return "Нет";
		}

		return "-";
	}

	$effect(() => {
		if (!browser) {
			return;
		}

		void loadObject();
	});
</script>

<div class="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-6 py-2 pb-8">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<Button href={resolve("/data")} variant="outline" class="rounded-full">
			<ChevronLeftIcon class="size-4" />
			К списку
		</Button>

		<div class="flex flex-wrap items-center gap-2">
			<Button
				variant="outline"
				class="rounded-full"
				disabled={isReprocessing || isLoading || !object}
				onclick={() => void handleReprocess()}
			>
				{#if isReprocessing}
					<LoaderIcon class="size-4 animate-spin" />
				{:else}
					<FileSearchIcon class="size-4" />
				{/if}
				Пере-OCR
			</Button>

			<Button
				class="rounded-full"
				disabled={isDownloading || isLoading || !object}
				onclick={() => void handleDownload()}
			>
				{#if isDownloading}
					<LoaderIcon class="size-4 animate-spin" />
				{:else}
					<DownloadIcon class="size-4" />
				{/if}
				Скачать файл
			</Button>
		</div>
	</div>

	{#if errorMessage}
		<Alert.Root variant="destructive">
			<Alert.Description>{errorMessage}</Alert.Description>
		</Alert.Root>
	{/if}

	{#if successMessage}
		<Alert.Root>
			<Alert.Description>{successMessage}</Alert.Description>
		</Alert.Root>
	{/if}

	{#if isLoading}
		<div class="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(20rem,1fr)]">
			<div class="space-y-6">
				<div class="rounded-3xl border p-6">
					<Skeleton class="mb-4 h-8 w-2/3 rounded-full" />
					<Skeleton class="mb-3 h-5 w-40 rounded-full" />
					<Skeleton class="h-24 w-full rounded-2xl" />
				</div>
				<div class="rounded-3xl border p-6">
					<Skeleton class="mb-4 h-6 w-32 rounded-full" />
					<Skeleton class="h-48 w-full rounded-2xl" />
				</div>
			</div>
			<div class="rounded-3xl border p-6">
				<Skeleton class="mb-4 h-6 w-28 rounded-full" />
				<div class="space-y-3">
					{#each [1, 2, 3, 4, 5, 6] as item (item)}
						<Skeleton class="h-10 w-full rounded-2xl" />
					{/each}
				</div>
			</div>
		</div>
	{:else if object}
		<div class="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(20rem,1fr)]">
			<div class="space-y-6">
				<section class="rounded-3xl border bg-card/70 p-6 shadow-sm">
					<div class="flex flex-col gap-4">
						<div class="flex flex-wrap items-center gap-3">
							<h1 class="text-2xl font-semibold tracking-tight">{getObjectTitle(object)}</h1>
							<DataStatusBadge status={object.status} />
							<Badge variant="outline">{getObjectTypeLabel(object)}</Badge>
						</div>

						{#if object.errorMessage}
							<p class="text-destructive text-sm leading-6">{object.errorMessage}</p>
						{/if}

						{#if object.tags?.length}
							<div class="flex flex-wrap gap-2">
								{#each object.tags as tag (tag)}
									<Badge variant="secondary">{tag}</Badge>
								{/each}
							</div>
						{/if}

						<div class="grid gap-4 text-sm text-muted-foreground sm:grid-cols-2 xl:grid-cols-3">
							<div class="rounded-2xl border px-4 py-3">
								<div class="text-xs uppercase tracking-wide">Размер</div>
								<div class="mt-1 text-foreground">{formatBytes(object.size)}</div>
							</div>
							<div class="rounded-2xl border px-4 py-3">
								<div class="text-xs uppercase tracking-wide">Создан</div>
								<div class="mt-1 text-foreground">{formatDateTime(object.createdAt)}</div>
							</div>
							<div class="rounded-2xl border px-4 py-3">
								<div class="text-xs uppercase tracking-wide">Обновлён</div>
								<div class="mt-1 text-foreground">{formatDateTime(object.updatedAt)}</div>
							</div>
						</div>
					</div>
				</section>

				<section class="rounded-3xl border bg-card/70 p-6 shadow-sm">
					<div class="mb-4 flex items-center justify-between gap-3">
						<h2 class="text-lg font-semibold">Содержимое</h2>
						{#if contentPreview.truncated}
							<span class="text-muted-foreground text-sm">Показан фрагмент</span>
						{/if}
					</div>

					{#if contentPreview.text}
						<pre class="bg-muted/50 overflow-x-auto rounded-2xl border p-4 text-sm leading-6 whitespace-pre-wrap">{contentPreview.text}</pre>
					{:else}
						<p class="text-muted-foreground text-sm">Текстовое содержимое пока недоступно.</p>
					{/if}
				</section>

				{#if metadataEntries.length > 0}
					<section class="rounded-3xl border bg-card/70 p-6 shadow-sm">
						<h2 class="mb-4 text-lg font-semibold">Дополнительные метаданные</h2>
						<div class="grid gap-3 sm:grid-cols-2">
							{#each metadataEntries as [key, value] (`${key}:${value}`)}
								<div class="rounded-2xl border px-4 py-3">
									<div class="text-muted-foreground text-xs uppercase tracking-wide">{key}</div>
									<div class="mt-1 break-words text-sm">{value}</div>
								</div>
							{/each}
						</div>
					</section>
				{/if}
			</div>

			<aside class="rounded-3xl border bg-card/70 p-6 shadow-sm">
				<h2 class="mb-4 text-lg font-semibold">Системные поля</h2>
				<dl class="space-y-4 text-sm">
					<div>
						<dt class="text-muted-foreground">ID</dt>
						<dd class="mt-1 break-all font-mono text-xs">{object.id}</dd>
					</div>
					<div>
						<dt class="text-muted-foreground">Имя файла</dt>
						<dd class="mt-1 break-words">{object.originalFilename || object.filename}</dd>
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
						<dt class="text-muted-foreground">Размер в байтах</dt>
						<dd class="mt-1">{object.sizeBytes ?? "-"}</dd>
					</div>
					<div>
						<dt class="text-muted-foreground">SHA-256</dt>
						<dd class="mt-1 break-all font-mono text-xs">{object.sha256 || "-"}</dd>
					</div>
					<div>
						<dt class="text-muted-foreground">Язык</dt>
						<dd class="mt-1">{object.language || "-"}</dd>
					</div>
					<div>
						<dt class="text-muted-foreground">Есть исходный файл</dt>
						<dd class="mt-1">{getBooleanLabel(object.hasContent)}</dd>
					</div>
					<div>
						<dt class="text-muted-foreground">Есть результат</dt>
						<dd class="mt-1">{getBooleanLabel(object.hasResult)}</dd>
					</div>
				</dl>
			</aside>
		</div>
	{:else}
		<div class="rounded-3xl border border-dashed p-8 text-center text-sm text-muted-foreground">
			Документ не найден.
		</div>
	{/if}
</div>
