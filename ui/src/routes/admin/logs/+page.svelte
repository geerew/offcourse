<script lang="ts">
	import { GetLogs } from '$lib/api/log-api';
	import { Spinner } from '$lib/components';
	import { ArrowDownToLineIcon, MoreInformationIcon, WarningIcon } from '$lib/components/icons';
	import LogLevelFilter from '$lib/components/pages/admin/logs/log-level.svelte';
	import LogTypeFilter from '$lib/components/pages/admin/logs/log-component.svelte';
	import { Badge, Button, Dialog } from '$lib/components/ui';
	import type { LogModel } from '$lib/models/log-model';
	import { cn } from '$lib/utils';
	import { Separator } from 'bits-ui';

	let logs: LogModel[] = $state([]);
	let currentPage = $state(1);
	let loading = $state(true);
	let selectedLevels = $state<string[]>([]);
	let levelFilter = $state('');
	let selectedTypes = $state<string[]>([]);
	let typeFilter = $state('');
	let showScrollToBottom = $state(false);
	let showScrollToTop = $state(false);
	let filtersInitialized = $state(false);
	let selectedLog: LogModel | null = $state(null);
	let dialogOpen = $state(false);

	const perPage = 50;

	let loadPromise = $state(fetchLogs());

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Fetch logs
	async function fetchLogs(): Promise<void> {
		try {
			loading = true;

			// Build level filters with OR condition
			let filterParts: string[] = [];
			if (levelFilter) {
				// Only wrap in parentheses if there are multiple OR conditions
				const levelFilterStr = levelFilter.includes(' OR ') ? `(${levelFilter})` : levelFilter;
				filterParts.push(levelFilterStr);
			}
			if (typeFilter) {
				// Only wrap in parentheses if there are multiple OR conditions
				const typeFilterStr = typeFilter.includes(' OR ') ? `(${typeFilter})` : typeFilter;
				filterParts.push(typeFilterStr);
			}

			const q = filterParts.length > 0 ? filterParts.join(' ') : undefined;

			const data = await GetLogs({
				q,
				page: currentPage,
				perPage
			});

			logs = data.items;
		} catch (error) {
			throw error;
		} finally {
			loading = false;
		}
	}

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Build level filter string from selectedLevels
	$effect(() => {
		levelFilter = selectedLevels.length ? selectedLevels.map((v) => `level:${v}`).join(' OR ') : '';
	});

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Build type filter string from selectedTypes
	$effect(() => {
		typeFilter = selectedTypes.length
			? selectedTypes.map((v) => `component:${v}`).join(' OR ')
			: '';
	});

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Fetch logs when filters change
	$effect(() => {
		// Skip the initial run (handled by loadPromise initialization)
		if (!filtersInitialized) {
			filtersInitialized = true;
			return;
		}

		// This effect runs when levelFilter or typeFilter changes
		// Reset to page 1 and fetch logs
		currentPage = 1;
		loadPromise = fetchLogs();
	});

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Get level display info
	function getLevelInfo(level: string) {
		switch (level) {
			case 'debug':
				return { label: 'debug', color: 'bg-background-alt-4 text-foreground-alt-2' };
			case 'info':
				return { label: 'info', color: 'bg-background-primary-alt-2 text-foreground-alt-6' };
			case 'warn':
				return { label: 'warn', color: 'bg-background-warning text-foreground-alt-1' };
			case 'error':
				return { label: 'error', color: 'bg-background-error text-foreground-alt-2' };
			default:
				return { label: level || 'unknown', color: 'bg-background-alt-4 text-foreground-alt-2' };
		}
	}

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Get component from data
	function getComponent(data: unknown): string | null {
		if (data && typeof data === 'object' && 'component' in data) {
			return String(data.component);
		}
		return null;
	}

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Get prioritized data fields
	function getPrioritizedData(
		data: unknown,
		component: string | null
	): Array<{ key: string; value: string; isError?: boolean }> {
		if (!data || typeof data !== 'object' || data === null) {
			return [];
		}

		const allEntries: Array<{ key: string; value: string; isError?: boolean }> = [];
		const other: Array<{ key: string; value: string; isError?: boolean }> = [];

		// Process all data fields
		for (const [key, value] of Object.entries(data)) {
			if (key === 'component') continue;

			let displayValue = String(value);

			// Format duration as milliseconds (rounded)
			if (key === 'duration' && typeof value === 'number') {
				displayValue = `${Math.round(value)}ms`;
			}

			const entry = { key, value: displayValue, isError: key === 'error_message' };

			// Collect prioritized entries separately
			if (component === 'api') {
				if (key === 'status' || key === 'error_message') {
					allEntries.push(entry);
				} else {
					other.push(entry);
				}
			} else {
				if (key === 'error_message') {
					allEntries.push(entry);
				} else {
					other.push(entry);
				}
			}
		}

		// Sort prioritized entries: status first (if api), then error_message
		if (component === 'api') {
			allEntries.sort((a, b) => {
				if (a.key === 'status') return -1;
				if (b.key === 'status') return 1;
				if (a.key === 'error_message') return -1;
				if (b.key === 'error_message') return 1;
				return 0;
			});
		}

		// Return prioritized first, then others
		return [...allEntries, ...other];
	}

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Format timestamp
	function formatTimestamp(date: string): string {
		const d = new Date(date);
		return d.toLocaleString(undefined, {
			year: 'numeric',
			month: '2-digit',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit'
		});
	}

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Format log data as JSON
	function formatLogData(log: LogModel): string {
		const logData = {
			id: log.id,
			createdAt: log.createdAt,
			level: log.level,
			message: log.message,
			data: log.data
		};
		return JSON.stringify(logData, null, 2);
	}

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Open dialog for a specific log
	function openLogDialog(log: LogModel): void {
		selectedLog = log;
		dialogOpen = true;
	}

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Check scroll position to show/hide scroll to bottom button
	function checkScrollPosition(): void {
		const scrollTop = window.scrollY || document.documentElement.scrollTop;
		const scrollHeight = document.documentElement.scrollHeight;
		const clientHeight = window.innerHeight;

		// Show button if not at bottom (with 50px threshold)
		showScrollToBottom = scrollHeight - scrollTop - clientHeight > 50;

		// Show button if not at top (with 50px threshold)
		showScrollToTop = scrollTop > 50;
	}

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Scroll to bottom
	function scrollToBottom(): void {
		window.scrollTo({
			top: document.documentElement.scrollHeight,
			behavior: 'smooth'
		});
	}

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Scroll to top
	function scrollToTop(): void {
		window.scrollTo({
			top: 0,
			behavior: 'smooth'
		});
	}

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Set up scroll listener
	$effect(() => {
		window.addEventListener('scroll', checkScrollPosition);
		checkScrollPosition(); // Check initial position

		return () => {
			window.removeEventListener('scroll', checkScrollPosition);
		};
	});

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

	// Update scroll position when logs change
	$effect(() => {
		if (logs.length > 0) {
			// Check scroll position after logs are rendered
			setTimeout(() => checkScrollPosition(), 100);
		}
	});
</script>

<div class="flex w-full place-content-center">
	<div class="flex w-full max-w-7xl flex-col gap-6 pt-1">
		<div class="flex flex-row items-center gap-3">
			<LogLevelFilter bind:selected={selectedLevels} />
			<LogTypeFilter bind:selected={selectedTypes} />
		</div>

		<div class="relative flex w-full place-content-center">
			{#await loadPromise}
				<div class="flex justify-center pt-10">
					<Spinner class="bg-foreground-alt-3 size-4" />
				</div>
			{:then _}
				<div class="flex w-full flex-col gap-3">
					{#if logs.length === 0}
						<div class="flex w-full flex-col items-center gap-2 pt-10">
							<div>No logs found</div>
						</div>
					{:else}
						{#each logs.slice().reverse() as log (log.id)}
							{@const levelInfo = getLevelInfo(log.level)}
							{@const component = getComponent(log.data)}
							{@const extraData = getPrioritizedData(log.data, component)}
							<div
								class="border-background-alt-3 flex flex-col gap-4 border-b pb-3 last:border-b-0"
							>
								<!-- First line: Timestamp, Message, Level Badge, and More Info Button -->
								<div class="flex flex-row items-start gap-3">
									<!-- Timestamp -->
									<div
										class="text-foreground-alt-3 mt-[3px] shrink-0 font-mono text-xs whitespace-nowrap"
										title={log.createdAt}
									>
										{formatTimestamp(log.createdAt)}
									</div>

									<!-- Message -->
									<div class="text-foreground-alt-1 min-w-0 flex-1 text-sm wrap-break-word">
										{log.message}
									</div>

									<!-- Level Badge and More Info Button -->
									<div class="flex shrink-0 items-center justify-end gap-2">
										<Badge class={cn('text-xs font-medium lowercase', levelInfo.color)}>
											{levelInfo.label}
										</Badge>
										<Button variant="ghost" class="h-6 w-6 p-0" onclick={() => openLogDialog(log)}>
											<MoreInformationIcon class="text-foreground-alt-3 size-4" />
										</Button>
									</div>
								</div>

								<!-- Second line: Component Badge, Separator, and Extra Data -->
								<div class="flex flex-row flex-wrap items-center gap-2">
									<!-- Component Badge -->
									{#if component}
										<Badge
											class="bg-background-alt-3 text-foreground-alt-2 shrink-0 text-xs font-medium lowercase select-none"
										>
											{component}
										</Badge>
									{/if}

									<!-- Vertical Separator -->
									{#if component && extraData.length > 0}
										<Separator.Root
											class="bg-background-alt-3 h-4 w-px shrink-0"
											orientation="vertical"
										/>
									{/if}

									<!-- Extra Data -->
									{#if extraData.length > 0}
										{#each extraData as { key, value, isError }}
											<Badge
												class={cn(
													'max-w-xs min-w-0 overflow-hidden text-xs font-medium whitespace-normal select-none',
													isError
														? 'bg-background-error text-foreground-alt-2'
														: 'bg-background-alt-2 text-foreground-alt-3'
												)}
												title={`${key}: ${value}`}
											>
												<span class="inline-block max-w-full min-w-0 truncate">{key}: {value}</span>
											</Badge>
										{/each}
									{/if}
								</div>
							</div>
						{/each}
					{/if}
				</div>

				<!-- Scroll to Top Button -->
				{#if showScrollToTop}
					<Button
						variant="default"
						class="bg-background-primary-alt-2 fixed bottom-20 left-[calc(var(--settings-menu-width)+1rem)] z-100 size-7 rounded-md shadow-lg"
						onclick={scrollToTop}
					>
						<ArrowDownToLineIcon class="stroke-foreground-alt-6 size-4 rotate-180 duration-200" />
					</Button>
				{/if}

				<!-- Scroll to Bottom Button -->
				{#if showScrollToBottom}
					<Button
						variant="default"
						class="bg-background-primary-alt-2 fixed bottom-10 left-[calc(var(--settings-menu-width)+1rem)] z-100 size-7 rounded-md shadow-lg"
						onclick={scrollToBottom}
					>
						<ArrowDownToLineIcon class="stroke-foreground-alt-6 size-4 duration-200" />
					</Button>
				{/if}
			{:catch error}
				<div class="flex w-full flex-col items-center gap-2 pt-10">
					<WarningIcon class="text-foreground-error size-10" />
					<span class="text-lg">Failed to fetch logs: {error.message}</span>
				</div>
			{/await}
		</div>
	</div>
</div>

<!-- Log Details Dialog -->
{#if selectedLog}
	<Dialog.Root bind:open={dialogOpen}>
		<Dialog.Content class="max-w-3xl">
			<Dialog.Header>
				<span>Log Details</span>
			</Dialog.Header>

			<div class="bg-background-alt-2 overflow-auto rounded-md p-4">
				<pre
					class="text-foreground-alt-1 font-mono text-xs wrap-break-word whitespace-pre-wrap">{formatLogData(
						selectedLog
					)}</pre>
			</div>

			<Dialog.Footer>
				<Dialog.CloseButton>Close</Dialog.CloseButton>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>
{/if}
