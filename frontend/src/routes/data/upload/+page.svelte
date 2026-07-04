<script lang="ts">
	import JSZip from "jszip";
	import { onDestroy } from "svelte";
	import AnimatedList from "$lib/components/ui/AnimatedList.svelte";
	import { getApiErrorMessage } from "$lib/api/auth";
	import { uploadKnowledgeObjects } from "$lib/api/data";
	import { Button } from "$lib/components/ui/button/index.js";
	import UploadIcon from "@lucide/svelte/icons/upload";
	import { formatBytes } from "$lib/data/utils";

	type UploadStatus = "idle" | "uploading" | "success" | "error";

	type UploadEntry = {
		name: string;
		size: number;
		file: File;
		progress: number;
		status: UploadStatus;
		showProgress: boolean;
		errorMessage: string;
		startedAt: number | null;
		finishedAt: number | null;
	};

	let entries = $state<UploadEntry[]>([]);
	let inputRef: HTMLInputElement;
	let dragCounter = $state(0);
	let isUploading = $state(false);
	let uploadError = $state("");
	let isDragging = $derived(dragCounter > 0);
	let fileItems = $derived(entries.map((entry) => entry.name));
	let fileBadges = $derived(entries.map((entry) => getFileBadges(entry)));
	let fileProgresses = $derived(entries.map((entry) => (entry.showProgress ? entry.progress : undefined)));
	let fileProgressLabels = $derived(entries.map((entry) => (entry.showProgress ? getProgressLabel(entry) : undefined)));
	let fileProgressClasses = $derived(entries.map((entry) => (entry.showProgress ? getProgressBarClass(entry) : undefined)));
	let fileMessages = $derived(entries.map((entry) => entry.errorMessage));
	let fileSucceeded = $derived(entries.map((entry) => entry.status === "success"));
	let now = $state(Date.now());
	let uploadStartedAt = $state<number | null>(null);
	let uploadFinishedAt = $state<number | null>(null);
	let ticker: ReturnType<typeof setInterval> | null = null;
	let uploadButtonLabel = $derived(isUploading ? `Загрузка... ${formatElapsed(getTotalElapsedMs())}` : "Загрузить");

	const SUPPORTED_EXTENSIONS = [".pdf", ".docx", ".pptx"];

	function startTicker() {
		stopTicker();
		now = Date.now();
		ticker = setInterval(() => {
			now = Date.now();
		}, 100);
	}

	function stopTicker() {
		if (ticker) {
			clearInterval(ticker);
			ticker = null;
		}
	}

	onDestroy(() => {
		stopTicker();
	});

	function isSupported(name: string): boolean {
		const lower = name.toLowerCase();
		return SUPPORTED_EXTENSIONS.some((ext) => lower.endsWith(ext));
	}

	async function addFiles(fileList: FileList | File[]) {
		const newEntries: UploadEntry[] = [];

		for (const file of Array.from(fileList)) {
			if (file.name.toLowerCase().endsWith(".zip")) {
				const extracted = await extractZip(file);
				for (const f of extracted) {
					newEntries.push({ name: f.name, size: f.size, file: f, progress: 0, status: "idle", showProgress: false, errorMessage: "", startedAt: null, finishedAt: null });
				}
			} else if (isSupported(file.name)) {
				newEntries.push({ name: file.name, size: file.size, file, progress: 0, status: "idle", showProgress: false, errorMessage: "", startedAt: null, finishedAt: null });
			}
		}

		entries = [...entries, ...newEntries];
	}

	async function extractZip(zipFile: File): Promise<File[]> {
		const zip = new JSZip();
		const data = await zip.loadAsync(zipFile);
		const results: File[] = [];
		const tasks: Promise<void>[] = [];

		data.forEach((relativePath, entry) => {
			if (!entry.dir && isSupported(relativePath)) {
				tasks.push(
					entry.async("blob").then((blob) => {
						results.push(new File([blob], relativePath));
					}),
				);
			}
		});

		await Promise.all(tasks);
		return results;
	}

	function handleDragEnter(e: DragEvent) {
		e.preventDefault();
		dragCounter++;
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
	}

	function handleDragLeave(e: DragEvent) {
		e.preventDefault();
		dragCounter--;
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		dragCounter = 0;
		if (e.dataTransfer?.files) {
			addFiles(e.dataTransfer.files);
		}
	}

	function handleInputChange(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		if (input.files) {
			addFiles(input.files);
		}
		input.value = "";
	}

	function removeFile(index: number) {
		if (isUploading) {
			return;
		}

		entries = entries.filter((_, i) => i !== index);
	}

	function getFileBadges(entry: UploadEntry): string[] {
		const base = entry.name.split('/').pop() || entry.name;
		const dot = base.lastIndexOf('.');
		const fmt = dot > 0 ? base.slice(dot + 1).toUpperCase() : '';
		return [formatBytes(entry.size), fmt].filter(Boolean);
	}

	function getProgressLabel(entry: UploadEntry): string {
		const elapsed = formatElapsed(getEntryElapsedMs(entry));

		if (entry.status === "success") {
			return `Готово ${elapsed}`;
		}

		if (entry.status === "error") {
			return `Ошибка ${elapsed}`;
		}

		return elapsed;
	}

	function getEntryElapsedMs(entry: UploadEntry): number {
		if (!entry.startedAt) {
			return 0;
		}

		const end = entry.finishedAt ?? now;
		return Math.max(end - entry.startedAt, 0);
	}

	function getTotalElapsedMs(): number {
		if (!uploadStartedAt) {
			return 0;
		}

		const end = uploadFinishedAt ?? now;
		return Math.max(end - uploadStartedAt, 0);
	}

	function formatElapsed(ms: number): string {
		return `${(ms / 1000).toFixed(1)}s`;
	}

	function getProgressBarClass(entry: UploadEntry): string {
		if (entry.status === "success") {
			return "bg-emerald-400";
		}

		if (entry.status === "error") {
			return "bg-destructive";
		}

		return "bg-primary";
	}

	async function uploadEntry(entry: UploadEntry): Promise<void> {
		entry.startedAt = Date.now();
		entry.finishedAt = null;
		entry.status = "uploading";
		entry.showProgress = true;

		try {
			await uploadKnowledgeObjects(
				[entry.file],
				{ recursive: true },
				(progressEvent) => {
					if (!progressEvent.total) {
						return;
					}

					entry.progress = Math.min(
						100,
						Math.round((progressEvent.loaded / progressEvent.total) * 100),
					);
				},
			);
			entry.progress = 100;
			entry.status = "success";
			entry.finishedAt = Date.now();
		} catch (error) {
			entry.status = "error";
			entry.finishedAt = Date.now();
			entry.errorMessage = getApiErrorMessage(error, `Не удалось загрузить файл ${entry.name}.`);
			uploadError ||= entry.errorMessage;
		}
	}

	async function handleUpload(): Promise<void> {
		if (entries.length === 0 || isUploading) {
			return;
		}

		const entriesToUpload = entries.filter((entry) => entry.status !== "success");

		if (entriesToUpload.length === 0) {
			return;
		}

		isUploading = true;
		uploadError = "";
		uploadStartedAt = Date.now();
		uploadFinishedAt = null;
		startTicker();

		for (const entry of entries) {
			entry.showProgress = false;
			if (entriesToUpload.includes(entry)) {
				entry.progress = 0;
				entry.status = "idle";
				entry.errorMessage = "";
				entry.startedAt = null;
				entry.finishedAt = null;
			}
		}

		await Promise.all(entriesToUpload.map((entry) => uploadEntry(entry)));

		isUploading = false;
		uploadFinishedAt = Date.now();
		now = uploadFinishedAt;
		stopTicker();

		for (const entry of entriesToUpload) {
			entry.showProgress = false;
		}
	}
</script>

<div
	role="region"
	aria-label="Зона загрузки"
	class="relative flex h-full flex-col overflow-hidden transition-colors {isDragging ? 'bg-muted/10' : ''}"
	ondragenter={handleDragEnter}
	ondragover={handleDragOver}
	ondragleave={handleDragLeave}
	ondrop={handleDrop}
>
	<input
		bind:this={inputRef}
		type="file"
		multiple
		class="sr-only"
		onchange={handleInputChange}
	/>

	{#if uploadError}
		<div class="bg-destructive/10 text-destructive mx-4 mt-4 rounded-2xl border border-destructive/20 px-4 py-3 text-sm">
			{uploadError}
		</div>
	{/if}

	{#if entries.length === 0}
		<div
			class="flex flex-1 cursor-pointer items-center justify-center px-4"
			onclick={() => inputRef?.click()}
			role="button"
			tabindex="0"
			onkeydown={(e) => e.key === 'Enter' && inputRef?.click()}
		>
			<div class="text-muted-foreground flex flex-col items-center gap-4 text-center">
				<UploadIcon class="size-16 opacity-40" />
				<p class="text-lg">Нажмите чтобы выбрать файлы или перетащите их сюда</p>
				<p class="text-sm">PDF, DOCX, PPTX, а также ZIP-архивы</p>
			</div>
		</div>
	{:else}
		<div class="flex min-h-0 flex-1 flex-col">
			<AnimatedList
				items={fileItems}
				itemBadges={fileBadges}
				itemProgresses={fileProgresses}
				itemProgressLabels={fileProgressLabels}
				itemProgressClasses={fileProgressClasses}
				itemMessages={fileMessages}
				itemSucceeded={fileSucceeded}
				onRemove={removeFile}
				removeDisabled={isUploading}
				showGradients={false}
				displayScrollbar={true}
				class="h-full w-full"
				listClass="max-h-none flex-1"
			/>
		</div>
	{/if}

	<div class="fixed bottom-6 right-6 z-50">
		<Button
			type="button"
			class="h-12 rounded-full px-6 shadow-lg text-base font-medium"
			disabled={entries.length === 0 || isUploading}
			onclick={() => void handleUpload()}
		>
			{uploadButtonLabel}
		</Button>
	</div>
</div>
