<script lang="ts">
	import JSZip from "jszip";
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
		errorMessage: string;
	};

	let entries = $state<UploadEntry[]>([]);
	let inputRef: HTMLInputElement;
	let dragCounter = $state(0);
	let isUploading = $state(false);
	let uploadError = $state("");
	let isDragging = $derived(dragCounter > 0);
	let fileItems = $derived(entries.map((entry) => trimFileName(entry.name)));
	let fileBadges = $derived(entries.map((entry) => getFileBadges(entry)));
	let fileProgresses = $derived(entries.map((entry) => entry.progress));
	let fileProgressLabels = $derived(entries.map((entry) => getProgressLabel(entry)));
	let fileProgressClasses = $derived(entries.map((entry) => getProgressBarClass(entry)));
	let fileMessages = $derived(entries.map((entry) => entry.errorMessage));

	const SUPPORTED_EXTENSIONS = [".pdf", ".doc", ".docx", ".pptx"];

	function isSupported(name: string): boolean {
		const lower = name.toLowerCase();
		return SUPPORTED_EXTENSIONS.some((ext) => lower.endsWith(ext));
	}

	function trimFileName(name: string, maxLen: number = 40): string {
		// Use only the base name (last path segment) — zip entries carry full paths
		const base = name.split('/').pop() || name;
		// Strip extension — shown in the format badge
		const dot = base.lastIndexOf('.');
		const stem = dot > 0 ? base.slice(0, dot) : base;
		if (stem.length <= maxLen) return stem;
		return stem.slice(0, Math.max(maxLen - 3, 0)) + '...';
	}

	async function addFiles(fileList: FileList | File[]) {
		const newEntries: UploadEntry[] = [];

		for (const file of Array.from(fileList)) {
			if (file.name.toLowerCase().endsWith(".zip")) {
				const extracted = await extractZip(file);
				for (const f of extracted) {
					newEntries.push({ name: f.name, size: f.size, file: f, progress: 0, status: "idle", errorMessage: "" });
				}
			} else if (isSupported(file.name)) {
				newEntries.push({ name: file.name, size: file.size, file, progress: 0, status: "idle", errorMessage: "" });
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
		if (entry.status === "success") {
			return "Готово";
		}

		if (entry.status === "error") {
			return "Ошибка";
		}

		return `${entry.progress}%`;
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

	async function handleUpload(): Promise<void> {
		if (entries.length === 0 || isUploading) {
			return;
		}

		isUploading = true;
		uploadError = "";

		for (const entry of entries) {
			entry.progress = 0;
			entry.status = "idle";
			entry.errorMessage = "";
		}

		for (const entry of entries) {
			entry.status = "uploading";

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
			} catch (error) {
				entry.status = "error";
				entry.errorMessage = getApiErrorMessage(error, `Не удалось загрузить файл ${entry.name}.`);
				uploadError ||= entry.errorMessage;
			}
		}

		isUploading = false;
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
				<p class="text-sm">PDF, DOC, DOCX, PPTX, а также ZIP-архивы</p>
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
			{isUploading ? "Загрузка..." : "Загрузить"}
		</Button>
	</div>
</div>
