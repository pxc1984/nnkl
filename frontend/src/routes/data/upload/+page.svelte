<script lang="ts">
	import JSZip from "jszip";
	import { onDestroy } from "svelte";
	import AnimatedList from "$lib/components/ui/AnimatedList.svelte";
	import { getApiErrorMessage } from "$lib/api/auth";
	import { uploadKnowledgeObjects } from "$lib/api/data";
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
	let isPreparingFiles = $state(false);
	let filePreparationProgress = $state(0);
	let filePreparationLabel = $state("");
	let isUploading = $state(false);
	let uploadError = $state("");
	let isDragging = $derived(dragCounter > 0);
	let isBusy = $derived(isPreparingFiles || isUploading);
	let fileItems = $derived(entries.map((entry) => entry.name));
	let fileBadges = $derived(entries.map((entry) => getFileBadges(entry)));
	let fileProgresses = $derived(entries.map((entry) => (entry.showProgress ? entry.progress : undefined)));
	let fileProgressLabels = $derived(entries.map((entry) => (entry.showProgress ? getProgressLabel(entry) : undefined)));
	let fileProgressClasses = $derived(entries.map((entry) => (entry.showProgress ? getProgressBarClass(entry) : undefined)));
	let fileMessages = $derived(entries.map((entry) => entry.errorMessage));
	let fileSucceeded = $derived(entries.map((entry) => entry.status === "success"));
	let now = $state(Date.now());
	let ticker: ReturnType<typeof setInterval> | null = null;
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

	function setFilePreparationState(progress: number, label: string): void {
		filePreparationProgress = Math.max(0, Math.min(100, Math.round(progress)));
		filePreparationLabel = label;
	}

	function openFileDialog(): void {
		if (isBusy) {
			return;
		}

		inputRef?.click();
	}

	function readFileAsArrayBuffer(file: File, onProgress: (progress: number) => void): Promise<ArrayBuffer> {
		return new Promise((resolve, reject) => {
			const reader = new FileReader();

			reader.onprogress = (event) => {
				if (event.lengthComputable && event.total > 0) {
					onProgress(event.loaded / event.total);
				}
			};

			reader.onload = () => {
				onProgress(1);
				resolve(reader.result as ArrayBuffer);
			};

			reader.onerror = () => {
				reject(reader.error ?? new Error(`Не удалось прочитать файл ${file.name}.`));
			};

			reader.readAsArrayBuffer(file);
		});
	}

	async function addFiles(fileList: FileList | File[]) {
		if (isBusy) {
			return;
		}

		const files = Array.from(fileList);
		if (files.length === 0) {
			return;
		}

		const newEntries: UploadEntry[] = [];
		const totalBytes = Math.max(files.reduce((sum, file) => sum + file.size, 0), files.length);
		let processedBytes = 0;

		isPreparingFiles = true;
		setFilePreparationState(0, "Готовим файлы...");

		try {
			for (const file of files) {
				const baseProcessedBytes = processedBytes;
				const updateOverallProgress = (fileProgress: number, label: string) => {
					const progress = ((baseProcessedBytes + file.size * fileProgress) / totalBytes) * 100;
					setFilePreparationState(progress, label);
				};

				if (file.name.toLowerCase().endsWith(".zip")) {
					const extracted = await extractZip(file, updateOverallProgress);
					for (const f of extracted) {
						newEntries.push({ name: f.name, size: f.size, file: f, progress: 0, status: "idle", showProgress: false, errorMessage: "", startedAt: null, finishedAt: null });
					}
				} else if (isSupported(file.name)) {
					updateOverallProgress(1, `Подготавливаем ${file.name}`);
					newEntries.push({ name: file.name, size: file.size, file, progress: 0, status: "idle", showProgress: false, errorMessage: "", startedAt: null, finishedAt: null });
				}

				processedBytes += file.size;
			}

			entries = [...entries, ...newEntries];

			if (newEntries.length > 0) {
				queueMicrotask(() => {
					void handleUpload();
				});
			}
		} finally {
			isPreparingFiles = false;
			setFilePreparationState(0, "");
		}
	}

	function updateEntry(index: number, updater: (entry: UploadEntry) => UploadEntry): void {
		entries = entries.map((entry, entryIndex) => (entryIndex === index ? updater(entry) : entry));
	}

	async function extractZip(zipFile: File, onProgress: (progress: number, label: string) => void): Promise<File[]> {
		const zip = new JSZip();
		const zipBuffer = await readFileAsArrayBuffer(zipFile, (progress) => {
			onProgress(progress * 0.8, `Читаем архив ${zipFile.name}`);
		});
		onProgress(0.82, `Открываем архив ${zipFile.name}`);
		const data = await zip.loadAsync(zipBuffer);
		const results: File[] = [];
		const zipEntries = data.filter((relativePath, entry) => !entry.dir && isSupported(relativePath));
		const totalEntries = Math.max(zipEntries.length, 1);

		for (const [index, entry] of zipEntries.entries()) {
			const fileName = entry.name.split('/').pop() || entry.name;
			const blob = await entry.async("blob", (metadata) => {
				const extractionProgress = (index + metadata.percent / 100) / totalEntries;
				onProgress(0.8 + extractionProgress * 0.2, `Распаковываем ${fileName}`);
			});
			results.push(new File([blob], entry.name));
		}

		onProgress(1, `Подготовили ${zipFile.name}`);
		return results;
	}

	function handleDragEnter(e: DragEvent) {
		e.preventDefault();
		if (isBusy) {
			return;
		}
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
		if (isBusy) {
			return;
		}

		if (e.dataTransfer?.files) {
			void addFiles(e.dataTransfer.files);
		}
	}

	async function handleInputChange(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		try {
			if (input.files) {
				await addFiles(input.files);
			}
		} finally {
			input.value = "";
		}
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

	async function uploadEntry(index: number): Promise<void> {
		updateEntry(index, (entry) => ({
			...entry,
			startedAt: null,
			finishedAt: null,
			progress: 0,
			status: "uploading",
			showProgress: true,
			errorMessage: "",
		}));

		try {
			const file = entries[index]?.file;
			if (!file) {
				return;
			}

			await uploadKnowledgeObjects(
				[file],
				{ recursive: true },
				(progressEvent) => {
					const total = progressEvent.total;
					if (!total) {
						return;
					}

					const progress = Math.min(
						100,
						Math.round((progressEvent.loaded / total) * 100),
					);

					updateEntry(index, (entry) => ({
						...entry,
						progress,
						startedAt: entry.startedAt ?? (progress === 100 ? Date.now() : null),
					}));
				},
			);

			updateEntry(index, (entry) => ({
				...entry,
				progress: 100,
				startedAt: entry.startedAt ?? Date.now(),
				status: "success",
				finishedAt: Date.now(),
			}));
		} catch (error) {
			const entryName = entries[index]?.name || "файл";
			const errorMessage = getApiErrorMessage(error, `Не удалось загрузить файл ${entryName}.`);
			updateEntry(index, (entry) => ({
				...entry,
				status: "error",
				finishedAt: Date.now(),
				errorMessage,
			}));
			uploadError ||= errorMessage;
		}
	}

	async function handleUpload(): Promise<void> {
		if (entries.length === 0 || isBusy) {
			return;
		}

		const entriesToUploadIndices = entries.flatMap((entry, index) => (entry.status !== "success" ? [index] : []));

		if (entriesToUploadIndices.length === 0) {
			return;
		}

		isUploading = true;
		uploadError = "";
		startTicker();

		entries = entries.map((entry) =>
			entry.status === "success"
				? { ...entry, showProgress: false }
				: {
					...entry,
					progress: 0,
					status: "idle",
					showProgress: false,
					errorMessage: "",
					startedAt: null,
					finishedAt: null,
				},
		);

		await Promise.all(entriesToUploadIndices.map((index) => uploadEntry(index)));

		isUploading = false;
		now = Date.now();
		stopTicker();

		entries = entries.map((entry) => ({ ...entry, showProgress: false }));
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
		disabled={isBusy}
		onchange={handleInputChange}
	/>

	{#if entries.length === 0}
		<div
			class="flex flex-1 items-center justify-center px-4 {isBusy ? 'cursor-default' : 'cursor-pointer'}"
			onclick={openFileDialog}
			role="button"
			tabindex="0"
			aria-disabled={isBusy}
			onkeydown={(e) => e.key === 'Enter' && openFileDialog()}
		>
			{#if isPreparingFiles}
				<div class="flex w-full max-w-md flex-col gap-4 text-center">
					<p class="text-lg">{filePreparationLabel}</p>
					<div class="h-2 overflow-hidden rounded-full bg-white/10" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={filePreparationProgress} aria-label="Подготовка файлов">
						<div class="bg-primary h-full rounded-full transition-[width]" style:width={`${filePreparationProgress}%`}></div>
					</div>
					<p class="text-muted-foreground text-sm">{filePreparationProgress}%</p>
				</div>
			{:else}
				<div class="text-muted-foreground flex flex-col items-center gap-4 text-center">
					<UploadIcon class="size-16 opacity-40" />
					<p class="text-lg">Нажмите чтобы выбрать файлы или перетащите их сюда</p>
					<p class="text-sm">PDF, DOCX, PPTX, а также ZIP-архивы</p>
				</div>
			{/if}
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
</div>
