<script lang="ts">
	import { browser } from "$app/environment";
	import { getApiErrorMessage } from "$lib/api/auth";
	import { listKnowledgeObjects } from "$lib/api/data";
	import DataStatusBadge from "$lib/components/data/data-status-badge.svelte";
	import { Button, buttonVariants } from "$lib/components/ui/button/index.js";
	import { Skeleton } from "$lib/components/ui/skeleton/index.js";
	import * as Table from "$lib/components/ui/table/index.js";
	import {
		FlexRender,
		createSvelteTable,
		renderComponent,
		renderSnippet,
	} from "$lib/components/ui/data-table/index.js";
	import type {
		ColumnDef,
		PaginationState,
	} from "@tanstack/table-core";
	import { getCoreRowModel } from "@tanstack/table-core";
	import type { KnowledgeObject, PaginationMeta } from "$lib/data/types";
	import { formatBytes, formatDateTime, getObjectTitle, getObjectTypeLabel } from "$lib/data/utils";
	import ChevronLeftIcon from "@lucide/svelte/icons/chevron-left";
	import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
	import { cn } from "$lib/utils.js";
	import { createRawSnippet } from "svelte";

	const PAGE_SIZE = 20;

	let objects = $state<KnowledgeObject[]>([]);
	let paginationMeta = $state<PaginationMeta>({ page: 1, pageSize: PAGE_SIZE, total: 0, totalPages: 1 });
	let isLoading = $state(false);
	let errorMessage = $state("");
	let currentPage = $state(1);
	let pagination = $state<PaginationState>({ pageIndex: 0, pageSize: PAGE_SIZE });
	let requestRun = 0;

	async function loadData(pageNum: number): Promise<void> {
		const currentRun = ++requestRun;
		isLoading = true;
		errorMessage = "";

		try {
			const response = await listKnowledgeObjects({ page: pageNum, pageSize: PAGE_SIZE });
			if (currentRun !== requestRun) {
				return;
			}

			objects = response.items;
			paginationMeta = response.meta;
		} catch (error) {
			if (currentRun !== requestRun) {
				return;
			}

			errorMessage = getApiErrorMessage(error, "Не удалось загрузить список материалов.");
			objects = [];
		} finally {
			if (currentRun === requestRun) {
				isLoading = false;
			}
		}
	}

	$effect(() => {
		if (!browser) {
			return;
		}

		void loadData(currentPage);
	});

	const totalPages = $derived(paginationMeta.totalPages ?? 1);
	const totalItems = $derived(paginationMeta.total ?? 0);

	const titleSnippet = createRawSnippet<[{ title: string }]>((getTitle) => {
		const { title } = getTitle();
		return {
			render: () => `<span class="font-medium block truncate">${title}</span>`,
		};
	});

	const typeSnippet = createRawSnippet<[{ type: string }]>((getType) => {
		const { type } = getType();
		return {
			render: () => `<span class="text-muted-foreground">${type}</span>`,
		};
	});

	const sizeSnippet = createRawSnippet<[{ size: string }]>((getSize) => {
		const { size } = getSize();
		return {
			render: () => `<span class="text-muted-foreground">${size}</span>`,
		};
	});

	const dateSnippet = createRawSnippet<[{ date: string }]>((getDate) => {
		const { date } = getDate();
		return {
			render: () => `<span class="text-muted-foreground">${date}</span>`,
		};
	});

	const actionSnippet = createRawSnippet<[{ id: string }]>((getId) => {
		const { id } = getId();
		const classes = cn(
			buttonVariants({ variant: "outline", size: "sm" }),
			"rounded-full",
		);
		return {
			render: () => `<a href="/data/${id}" class="${classes}">Открыть</a>`,
		};
	});

	const columns: ColumnDef<KnowledgeObject>[] = [
		{
			accessorKey: "filename",
			header: "Название",
			cell: ({ row }) => renderSnippet(titleSnippet, { title: getObjectTitle(row.original) }),
		},
		{
			accessorKey: "size",
			header: "Размер",
			cell: ({ row }) => renderSnippet(sizeSnippet, { size: formatBytes(row.original.size) }),
		},
		{
			accessorKey: "status",
			header: "Статус",
			cell: ({ row }) => renderComponent(DataStatusBadge, { status: row.original.status }),
		},
		{
			accessorKey: "createdAt",
			header: "Загружен",
			cell: ({ row }) => renderSnippet(dateSnippet, { date: formatDateTime(row.original.createdAt) }),
		},
		{
			id: "actions",
			header: "",
			cell: ({ row }) => renderSnippet(actionSnippet, { id: row.original.id }),
		},
	];

	const table = createSvelteTable({
		get data() {
			return objects;
		},
		columns,
		getCoreRowModel: getCoreRowModel(),
		manualPagination: true,
		get pageCount() {
			return totalPages;
		},
		state: {
			get pagination() {
				return pagination;
			},
		},
		onPaginationChange: (updater) => {
			if (typeof updater === "function") {
				pagination = updater(pagination);
			} else {
				pagination = updater;
			}
			currentPage = pagination.pageIndex + 1;
		},
	});
</script>

<div class="flex flex-col justify-between h-full">
	{#if errorMessage}
		<div class="text-destructive bg-destructive/10 rounded-2xl border border-destructive/20 px-4 py-3 text-sm">
			{errorMessage}
		</div>
	{/if}

	{#if isLoading}
		<div class="rounded-md border p-6">
			{#each [1, 2, 3, 4, 5, 6] as i (i)}
				<div class="flex items-center gap-4 py-3">
					<Skeleton class="h-5 w-48 rounded-full" />
					<Skeleton class="h-5 w-20 rounded-full" />
					<Skeleton class="h-5 w-16 rounded-full" />
					<Skeleton class="h-5 w-20 rounded-full" />
					<Skeleton class="h-5 w-32 rounded-full" />
					<Skeleton class="h-5 w-28 rounded-full" />
					<Skeleton class="ms-auto h-8 w-24 rounded-full" />
				</div>
			{/each}
		</div>
	{:else}
		<div class="rounded-md border">
			<Table.Root class="table-fixed">
				<Table.Header>
					{#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
						<Table.Row>
							{#each headerGroup.headers as header (header.id)}
								<Table.Head class={cn("has-[[role=checkbox]]:ps-3", header.column.id === "mimeType" && "hidden md:table-cell w-24", header.column.id === "filename" && "w-full min-w-0", header.column.id === "size" && "w-20", header.column.id === "status" && "w-24", header.column.id === "createdAt" && "w-36", header.column.id === "actions" && "w-24")}>
									{#if !header.isPlaceholder}
										<FlexRender
											content={header.column.columnDef.header}
											context={header.getContext()}
										/>
									{/if}
								</Table.Head>
							{/each}
						</Table.Row>
					{/each}
				</Table.Header>
				<Table.Body>
					{#each table.getRowModel().rows as row (row.id)}
						<Table.Row>
							{#each row.getVisibleCells() as cell (cell.id)}
								<Table.Cell class={cn("has-[[role=checkbox]]:ps-3", cell.column.id === "mimeType" && "hidden md:table-cell", cell.column.id === "filename" && "min-w-0")}>
									<FlexRender
										content={cell.column.columnDef.cell}
										context={cell.getContext()}
									/>
								</Table.Cell>
							{/each}
						</Table.Row>
					{:else}
						<Table.Row>
							<Table.Cell colspan={columns.length} class="h-24 text-center">
								Нет результатов.
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		</div>

		<div class="flex items-center justify-between pt-4 bottom-4">
			<p class="text-muted-foreground text-sm">
				{totalItems === 1
					? "1 материал"
					: `${totalItems} материалов`}
			</p>

			<div class="flex items-center gap-2">
				<Button
					variant="outline"
					size="sm"
					class="rounded-full"
					disabled={!table.getCanPreviousPage()}
					onclick={() => table.previousPage()}
				>
					<ChevronLeftIcon class="size-4" />
				</Button>

				{#each Array.from({ length: Math.min(totalPages, 7) }, (__, i) => {
					const start = Math.max(0, Math.min(currentPage - 4, totalPages - 7));
					return start + i + 1;
				}) as pageNum (pageNum)}
					<Button
						variant={pageNum === currentPage ? "default" : "outline"}
						size="sm"
						class="rounded-full min-w-9"
						onclick={() => table.setPageIndex(pageNum - 1)}
					>
						{pageNum}
					</Button>
				{/each}

				<Button
					variant="outline"
					size="sm"
					class="rounded-full"
					disabled={!table.getCanNextPage()}
					onclick={() => table.nextPage()}
				>
					<ChevronRightIcon class="size-4" />
				</Button>
			</div>
		</div>
	{/if}
</div>