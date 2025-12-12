<script lang="ts">
	import { page } from '$app/state';
	import type { APIError } from '$lib/api-error.svelte';
	import {
		FavouriteCourse,
		GetCourse,
		GetCourseModules,
		GetCourseTags,
		UnfavouriteCourse,
		UpdateCourseAssetProgress
	} from '$lib/api/course-api';
	import { StartScan } from '$lib/api/scan-api';
	import { auth } from '$lib/auth.svelte';
	import { NiceDate, Spinner } from '$lib/components';
	import { ClearCourseProgressDialog, MarkCourseCompleteDialog } from '$lib/components/dialogs';
	import {
		AddedIcon,
		BookTextIcon,
		ClearProgressIcon,
		DotIcon,
		DotsIcon,
		DurationIcon,
		FavouriteIcon,
		HalfCircleIcon,
		FilesIcon,
		LoaderCircleIcon,
		ModulesIcon,
		PathIcon,
		PlayCircleIcon,
		ScanIcon,
		TagIcon,
		TickCircleIcon,
		TickIcon,
		UpdatedIcon,
		WarningIcon
	} from '$lib/components/icons';
	import { Badge, Checkbox, Dropdown } from '$lib/components/ui';
	import Attachments from '$lib/components/ui/attachments.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import type { AssetModel } from '$lib/models/asset-model';
	import type { AssetProgressUpdateModel } from '$lib/models/asset-model';
	import type { CourseModel, CourseReqParams, CourseTagsModel } from '$lib/models/course-model';
	import type { LessonModel, ModuleModel, ModulesModel } from '$lib/models/module-model';
	import type { ScanCreateModel } from '$lib/models/scan-model';
	import { scanStore } from '$lib/scanStore.svelte';
	import { cn } from '$lib/utils';
	import { useId } from 'bits-ui';
	import prettyMs from 'pretty-ms';
	import { toast } from 'svelte-sonner';

	// Course data
	let course = $state<CourseModel>();
	let modules = $state<ModulesModel>();
	let tags = $state<CourseTagsModel>([]);
	let loadPromise = $state(fetcher());

	// Dialog state
	let openCourseProgressDialog = $state(false);
	let openMarkCompleteDialog = $state(false);
	let dropdownOpen = $state(false);

	// Asset edit mode
	let isAssetEditMode = $state(false);
	let selectedAssets = $state<Record<string, AssetModel>>({});
	let isPostingAssets = $state(false);

	// Scan tracking
	let hasActiveScan = $state(false);
	let scanStatus = $derived.by(() => {
		if (!course) return undefined;
		return scanStore.getScanStatus(course.id);
	});
	let isScanning = $derived(scanStatus === 'processing' || scanStatus === 'waiting');

	// Progress tracking
	let isMarkingComplete = $state(false);
	let frozenProgress = $state<number | null>(null);
	let previousProgress = $state(0);
	let isCourseComplete = $derived(course?.progress?.percent === 100);
	let hasProgress = $derived(
		(course?.progress?.started ?? false) || (course?.progress?.percent ?? 0) > 0
	);
	let shouldAnimateProgress = $derived.by(() => {
		// Don't animate during mark complete operation
		if (isMarkingComplete) return false;

		const currentProgress = course?.progress?.percent ?? 0;
		// Only animate if progress is increasing or staying the same
		return currentProgress >= previousProgress;
	});

	// Course statistics
	let moduleCount = $derived(modules ? modules.modules.length : 0);
	let lessonCount = $derived.by(() => {
		if (!modules) return 0;
		let count = 0;
		for (const m of modules.modules) {
			count += m.lessons.length;
		}
		return count;
	});
	let assetCount = $derived.by(() => {
		if (!modules) return 0;
		let count = 0;
		for (const m of modules.modules) {
			for (const lesson of m.lessons) {
				count += lesson.assets.length;
			}
		}
		return count;
	});
	// First lesson to resume
	// Order: started-but-incomplete; then first incomplete; then first lesson
	let lessonToResume = $derived.by(() => {
		if (!modules) return null;

		// Started but not completed
		for (const mod of modules.modules) {
			for (const lesson of mod.lessons) {
				if (lesson.started && !lesson.completed) return lesson;
			}
		}

		// Any incomplete
		for (const mod of modules.modules) {
			for (const lesson of mod.lessons) {
				if (!lesson.completed) return lesson;
			}
		}

		// Fallback to first lesson
		return modules.modules[0]?.lessons[0] ?? null;
	});

	// Utilities
	const labelId = useId();
	const pad2 = (n: number) => String(n).padStart(2, '0');

	// Loads course data, tags, and modules
	async function fetcher(): Promise<void> {
		try {
			if (!page.params.course_id) throw new Error('No course ID provided');

			const courseReqParams: CourseReqParams = { withUserProgress: true };
			course = await GetCourse(page.params.course_id, courseReqParams);

			tags = await GetCourseTags(course.id);

			const moduleReqParams: CourseReqParams = { withUserProgress: true };
			modules = await GetCourseModules(course.id, moduleReqParams);
		} catch (error) {
			console.error('Error loading course page:', error);
			throw error;
		}
	}

	// Refreshes course and module data
	async function refreshCourse(): Promise<void> {
		if (!course) return;

		try {
			const courseReqParams: CourseReqParams = { withUserProgress: true };
			const refreshedCourse = await GetCourse(course.id, courseReqParams);
			course = refreshedCourse;

			// Also refresh modules if they exist
			if (modules) {
				const moduleReqParams: CourseReqParams = { withUserProgress: true };
				modules = await GetCourseModules(course.id, moduleReqParams);
			}
		} catch (error) {
			console.error('Failed to refresh course:', error);
		}
	}

	// Starts a scan for the course
	async function doScan(): Promise<void> {
		if (!course) return;

		try {
			await StartScan({ courseId: course.id } satisfies ScanCreateModel);
			toast.success('Scan started');
		} catch (error) {
			toast.error('Failed to start the scan: ' + (error as APIError).message);
		}
	}

	// Toggles favourite status
	async function toggleFavourite(): Promise<void> {
		if (!course) return;

		const wasFavourited = course.favourited ?? false;

		// Optimistically update UI
		course.favourited = !wasFavourited;

		try {
			if (wasFavourited) {
				await UnfavouriteCourse(course.id);
				toast.success('Course unfavourited');
			} else {
				await FavouriteCourse(course.id);
				toast.success('Course favourited');
			}
		} catch (error) {
			// Revert on error
			course.favourited = wasFavourited;
			toast.error(
				`Failed to ${wasFavourited ? 'unfavourite' : 'favourite'} course: ${(error as APIError).message}`
			);
		}
	}

	// Marks all assets as complete
	async function markCourseComplete(): Promise<void> {
		if (!course || !modules) return;

		isMarkingComplete = true;
		// Freeze the current progress value
		frozenProgress = course.progress?.percent ?? 0;

		try {
			// Mark all assets as completed
			const promises: Promise<void>[] = [];

			for (const mod of modules.modules) {
				for (const lesson of mod.lessons) {
					for (const asset of lesson.assets) {
						// Skip if already completed
						if (asset.progress.completed) continue;

						const progress: AssetProgressUpdateModel = {
							completed: true
						};

						promises.push(UpdateCourseAssetProgress(course.id, lesson.id, asset.id, progress));

						// Update local state
						asset.progress.completed = true;
						lesson.started = true;
						lesson.completed = true;
						lesson.assetsCompleted = lesson.assets.length;
					}
				}
			}

			await Promise.all(promises);

			// Small delay to allow backend to recalculate progress
			await new Promise((resolve) => setTimeout(resolve, 300));

			// Refresh course data to get updated progress
			await refreshCourse();

			// Ensure progress shows 100% after refresh
			if (course.progress) {
				course.progress.percent = 100;
			}

			toast.success('Course marked as complete');
		} catch (error) {
			toast.error('Failed to mark course as complete: ' + (error as APIError).message);
		} finally {
			// Small delay before unfreezing to ensure smooth transition
			await new Promise((resolve) => setTimeout(resolve, 50));
			frozenProgress = null;
			isMarkingComplete = false;
		}
	}

	// Returns all assets in a module
	function getAllAssetsInModule(modulePrefix: number): AssetModel[] {
		if (!modules) return [];
		const mod = modules.modules.find((m: ModuleModel) => m.prefix === modulePrefix);
		if (!mod) return [];

		const assets: AssetModel[] = [];
		for (const lesson of mod.lessons) {
			assets.push(...lesson.assets);
		}
		return assets;
	}

	// Toggles selection of all assets in a module
	function toggleModule(modulePrefix: number): void {
		if (!modules) return;

		const moduleAssets = getAllAssetsInModule(modulePrefix);
		const allSelected = moduleAssets.every((a) => selectedAssets[a.id]);

		// Create a new object to ensure reactivity
		const newSelectedAssets = { ...selectedAssets };

		if (allSelected) {
			// Deselect all assets in module
			moduleAssets.forEach((asset) => {
				delete newSelectedAssets[asset.id];
			});
		} else {
			// Select all assets in module
			moduleAssets.forEach((asset) => {
				newSelectedAssets[asset.id] = asset;
			});
		}

		selectedAssets = newSelectedAssets;
	}

	// Toggles selection of lesson assets
	function toggleLessonAssets(lessonAssets: AssetModel[]): void {
		const allSelected = lessonAssets.every((a) => selectedAssets[a.id]);
		const newSelectedAssets = { ...selectedAssets };

		lessonAssets.forEach((asset) => {
			if (allSelected) {
				delete newSelectedAssets[asset.id];
			} else {
				newSelectedAssets[asset.id] = asset;
			}
		});

		selectedAssets = newSelectedAssets;
	}

	// Enters asset edit mode and pre-selects completed assets
	function enterAssetEditMode(): void {
		if (!modules) return;

		isAssetEditMode = true;
		selectedAssets = {};

		// Pre-select completed assets
		for (const mod of modules.modules) {
			for (const lesson of mod.lessons) {
				for (const asset of lesson.assets) {
					if (asset.progress.completed) {
						selectedAssets[asset.id] = asset;
					}
				}
			}
		}
	}

	// Cancels asset edit mode
	function cancelAssetEditMode(): void {
		isAssetEditMode = false;
		selectedAssets = {};
	}

	// Saves asset completion changes
	async function confirmAssetChanges(): Promise<void> {
		if (!course || !modules) return;

		isPostingAssets = true;

		try {
			const promises: Promise<void>[] = [];
			const selectedAssetIds = new Set(Object.keys(selectedAssets));

			// Update all assets based on selection
			for (const mod of modules.modules) {
				for (const lesson of mod.lessons) {
					for (const asset of lesson.assets) {
						const shouldBeCompleted = selectedAssetIds.has(asset.id);
						const isCurrentlyCompleted = asset.progress.completed;

						// Only update if state changed
						if (shouldBeCompleted !== isCurrentlyCompleted) {
							const progress: AssetProgressUpdateModel = {
								completed: shouldBeCompleted
							};

							promises.push(UpdateCourseAssetProgress(course.id, lesson.id, asset.id, progress));

							// Update local state
							asset.progress.completed = shouldBeCompleted;
						}
					}

					// Update lesson completion state
					const completedAssets = lesson.assets.filter(
						(a: AssetModel) => a.progress.completed
					).length;
					lesson.assetsCompleted = completedAssets;
					lesson.completed = completedAssets === lesson.assets.length;
					lesson.started = completedAssets > 0;
				}
			}

			await Promise.all(promises);

			// Refresh course data to get updated progress
			await refreshCourse();

			toast.success('Asset status updated');
			isAssetEditMode = false;
			selectedAssets = {};
		} catch (error) {
			toast.error('Failed to update assets: ' + (error as APIError).message);
		}

		isPostingAssets = false;
	}

	// Register with scanStore
	$effect(() => {
		return scanStore.register();
	});

	// Watch for scan updates and refresh course when scan finishes
	$effect(() => {
		if (!course) return;

		const currentlyHasScan = isScanning;

		// If scan finished (had scan before, but no longer has one), refresh course
		if (hasActiveScan && !currentlyHasScan) {
			refreshCourse();
		}

		hasActiveScan = currentlyHasScan;
	});

	// Update previous progress when course progress changes (but not during mark complete)
	$effect(() => {
		if (!isMarkingComplete) {
			const currentProgress = course?.progress?.percent ?? 0;
			previousProgress = currentProgress;
		}
	});
</script>

{#await loadPromise}
	<div class="flex justify-center pt-10">
		<Spinner class="bg-foreground-alt-3 size-4" />
	</div>
{:then _}
	{#if course}
		<div class="flex w-full flex-col">
			<div class="flex w-full place-content-center">
				<div class="container-px flex w-full max-w-7xl flex-col gap-6 pt-5 pb-10 lg:pt-10">
					<div class="grid w-full grid-cols-1 gap-6 lg:grid-cols-[1fr_minmax(0,23rem)] lg:gap-10">
						<!-- Information -->
						<div class="order-2 flex h-full w-full flex-col justify-between gap-5 lg:order-1">
							<div class="flex h-full w-full flex-col gap-4 py-2">
								<!-- Title -->
								<div class="text-foreground-alt-1 text-lg font-semibold md:text-2xl">
									{course.title}
								</div>

								<!-- Status -->
								{#if isScanning || !course.available || course.maintenance || (course.initialScan !== undefined && !course.initialScan)}
									<div class="flex h-7 flex-col gap-x-3 gap-y-3 text-sm sm:flex-row">
										<div class="flex flex-row items-center gap-2">
											{#if isScanning && course.initialScan === false}
												<Badge class="bg-background-warning text-foreground-alt-1"
													>Initial Scan</Badge
												>
											{:else if isScanning}
												<Badge class="bg-background-warning text-foreground-alt-1"
													>Maintenance</Badge
												>
											{:else if !course.initialScan}
												<Badge class="bg-background-warning text-foreground-alt-1"
													>Initial Scan</Badge
												>
											{:else if course.maintenance}
												<Badge class="bg-background-warning text-foreground-alt-1"
													>Maintenance</Badge
												>
											{:else}
												<Badge class="bg-background-error text-foreground-alt-1">Unavailable</Badge>
											{/if}
										</div>
									</div>
								{/if}

								<!-- Overview -->
								<div class="flex flex-col gap-x-3 gap-y-3 text-sm sm:flex-row">
									<div class="flex flex-row items-center gap-2 font-semibold">
										<ModulesIcon class="text-foreground-alt-3 size-4.5" />
										<span class="text-nowrap">
											{moduleCount} module{moduleCount != 1 ? 's' : ''}
										</span>
									</div>

									<DotIcon class="text-foreground-alt-3 hidden text-xl sm:inline" />

									<div class="flex flex-row items-center gap-2 font-semibold">
										<FilesIcon class="text-foreground-alt-3 size-4.5" />
										<span class="text-nowrap">
											{lessonCount} lesson{lessonCount != 1 ? 's' : ''}
										</span>
									</div>

									<DotIcon class="text-foreground-alt-3 hidden text-xl sm:inline" />

									<div
										class="flex basis-full flex-row items-center gap-2 font-semibold sm:basis-auto"
									>
										<DurationIcon class="text-foreground-alt-3 size-4.5" />
										<span
											class={cn(
												'text-nowrap',
												course.duration ? 'text-foreground-alt-1' : 'text-foreground-alt-3'
											)}
										>
											{course.duration
												? prettyMs(course.duration * 1000, { hideSeconds: true })
												: '-'}
										</span>
									</div>
								</div>

								<!-- Progress Bar -->
								{#if course.progress?.started}
									{@const displayProgress =
										frozenProgress !== null ? frozenProgress : (course.progress?.percent ?? 0)}
									<div class="flex h-7 flex-row items-center gap-2">
										<LoaderCircleIcon class="text-foreground-alt-3 size-4.5" />

										<div
											class="bg-background-alt-3 relative h-5 w-full max-w-56 overflow-hidden rounded-md"
											aria-labelledby={labelId}
											role="progressbar"
											aria-valuenow={displayProgress}
											aria-valuemin="0"
											aria-valuemax="100"
										>
											<div
												class={cn(
													'bg-background-primary-alt-1/70 h-full',
													shouldAnimateProgress && frozenProgress === null
														? 'transition-all duration-1000 ease-in-out'
														: ''
												)}
												style={`width: ${displayProgress}%`}
											></div>

											<div
												id={labelId}
												class="text-foreground-alt-1 absolute inset-0 flex items-center justify-center text-xs font-medium drop-shadow-sm"
											>
												{displayProgress}%
											</div>
										</div>
									</div>
								{/if}

								<!-- Path -->
								{#if auth.user?.role === 'admin'}
									<div
										class="text-foreground-alt-2 flex flex-row items-start gap-2 text-sm leading-7"
									>
										<div class="mt-1">
											<PathIcon class="text-foreground-alt-3 size-4.5 shrink-0" />
										</div>

										<span class="wrap-anywhere whitespace-normal" title={course.path}
											>{course.path}</span
										>
									</div>
								{/if}

								<!-- Created/updated -->
								<div
									class="text-foreground-alt-2 flex flex-col gap-x-3 gap-y-3 text-sm sm:flex-row"
								>
									<div class="flex flex-row items-center gap-2">
										<AddedIcon class="text-foreground-alt-3 size-4.5" />
										<span>
											<NiceDate date={course.createdAt} prefix="Added:" />
										</span>
									</div>

									<DotIcon class="text-foreground-alt-3 hidden text-xl sm:inline" />

									<div class="flex flex-row items-center gap-2">
										<UpdatedIcon class="text-foreground-alt-3 size-4.5" />
										<span>
											<NiceDate date={course.updatedAt} prefix="Updated:" />
										</span>
									</div>
								</div>

								<!-- Description -->
								<div class="flex flex-col gap-3 text-sm">
									<div class="flex flex-row items-center gap-2">
										<BookTextIcon class="text-foreground-alt-3 size-4.5" />
										<span>Description</span>
									</div>
									{#if !course.description}
										<span class="text-foreground-alt-2 px-2">-</span>
									{:else}
										<div class="text-foreground-alt-2 pl-0.5 leading-relaxed">
											{course.description}
										</div>
									{/if}
								</div>

								<!-- Tags -->
								<div class="flex flex-col gap-3 text-sm">
									<div class="flex flex-row items-center gap-2">
										<TagIcon class="text-foreground-alt-3 size-4.5 stroke-2" />
										<span>Tags</span>
									</div>
									{#if tags.length === 0}
										<span class="text-foreground-alt-2 px-2">-</span>
									{:else}
										<div class="flex flex-wrap gap-2 pl-0.5">
											{#each tags as tag}
												<Badge class="text-sm  select-none">
													{tag.tag}
												</Badge>
											{/each}
										</div>
									{/if}
								</div>
							</div>

							{#if assetCount > 0}
								<div class="flex flex-row place-items-end gap-2.5 pt-3">
									<Button
										href={`/course/${course.id}/${lessonToResume?.id}`}
										variant="default"
										class="w-auto px-4"
										disabled={isScanning ||
											course.maintenance ||
											!course.available ||
											isAssetEditMode}
										onclick={(e) => {
											if (
												isScanning ||
												course?.maintenance ||
												!course?.available ||
												isAssetEditMode
											) {
												e.preventDefault();
												e.stopPropagation();
											}
										}}
									>
										{#if course.progress?.started}
											Resume Course
										{:else}
											Start Course
										{/if}
									</Button>

									<Button
										variant="ghost"
										class={cn(
											'bg-background-alt-3 data-[state=open]:bg-background-alt-4 hover:bg-background-alt-4 w-auto rounded-lg border-none'
										)}
										disabled={isAssetEditMode || isScanning}
										onclick={(e: MouseEvent) => {
											if (isAssetEditMode || isScanning) {
												e.preventDefault();
												e.stopPropagation();
												return;
											}
											toggleFavourite();
										}}
									>
										<FavouriteIcon
											class={cn(
												'size-5 stroke-[1.5]',
												course.favourited && 'fill-foreground-error text-foreground-error'
											)}
										/>
									</Button>

									<Dropdown.Root bind:open={dropdownOpen}>
										<Dropdown.Trigger
											class="bg-background-alt-3 data-[state=open]:bg-background-alt-4 hover:bg-background-alt-4 w-auto rounded-lg border-none"
											disabled={isAssetEditMode || isScanning}
											onclick={(e: MouseEvent) => {
												if (isAssetEditMode || isScanning) {
													e.preventDefault();
													e.stopPropagation();
												}
											}}
										>
											<DotsIcon class="size-5 stroke-[1.5]" />
										</Dropdown.Trigger>

										<Dropdown.Content class="z-60 w-52" align="start">
											{#if auth.user?.role === 'admin'}
												<Dropdown.Item
													class="data-disabled:pointer-events-none"
													disabled={isScanning}
													onclick={async () => {
														if (isScanning) return;
														doScan();
													}}
												>
													<ScanIcon class="size-4 stroke-[1.5]" />
													<span>Scan</span>
												</Dropdown.Item>

												<Dropdown.Separator />
											{/if}

											<Dropdown.Item
												onclick={() => {
													dropdownOpen = false;
													enterAssetEditMode();
												}}
											>
												<FilesIcon class="size-4 stroke-[1.5]" />
												<span>Manage Assets</span>
											</Dropdown.Item>

											<Dropdown.Separator />

											<Dropdown.Item
												class="data-disabled:pointer-events-none"
												disabled={isCourseComplete ||
													isScanning ||
													course?.maintenance ||
													!course?.available}
												onclick={async () => {
													if (
														isCourseComplete ||
														isScanning ||
														course?.maintenance ||
														!course?.available
													)
														return;

													// If course has no progress, mark complete directly
													// Otherwise show confirmation dialog
													if (!hasProgress) {
														markCourseComplete();
													} else {
														openMarkCompleteDialog = true;
													}
												}}
											>
												<TickIcon class="size-4 stroke-[1.5]" />
												<span>Mark Course Complete</span>
											</Dropdown.Item>

											<Dropdown.CautionItem
												class="data-disabled:pointer-events-none"
												disabled={!course?.progress?.started}
												onclick={async () => {
													openCourseProgressDialog = true;
												}}
											>
												<ClearProgressIcon class="size-4 stroke-[1.5]" />
												<span>Clear Course Progress</span>
											</Dropdown.CautionItem>
										</Dropdown.Content>
									</Dropdown.Root>

									<ClearCourseProgressDialog
										bind:open={openCourseProgressDialog}
										{course}
										successFn={() => {
											// Clear course progress (local state)
											if (!course) return;

											course.progress = {
												percent: 0,
												startedAt: '',
												started: false,
												completedAt: ''
											};

											// Clear asset progress (local state)
											if (!modules) return;

											for (const mod of modules.modules) {
												for (const lesson of mod.lessons) {
													lesson.completed = false;
													lesson.started = false;
													lesson.assetsCompleted = 0;

													for (const asset of lesson.assets) {
														asset.progress = {
															position: 0,
															completed: false,
															completedAt: ''
														};
													}
												}
											}
										}}
									/>

									<MarkCourseCompleteDialog
										bind:open={openMarkCompleteDialog}
										{course}
										successFn={() => {
											markCourseComplete();
										}}
									/>
								</div>
							{/if}
						</div>

						<!-- Card -->
						<div class="relative order-1 flex h-50 w-full justify-center rounded-lg lg:order-2">
							<img
								src={`/api/courses/${course.id}/card?v=${course.cardHash || 'fallback'}`}
								alt={course.title}
								class="h-auto max-h-full w-auto max-w-full rounded-lg object-contain"
							/>
						</div>
					</div>
				</div>
			</div>

			<!-- Course Content -->
			<div class="bg-background flex w-full place-content-center">
				<div
					class="container-px flex w-full max-w-7xl flex-col"
					class:pb-24={isAssetEditMode}
					class:pb-10={!isAssetEditMode}
				>
					<div class="text-foreground-alt-1 flex flex-col gap-12 sm:gap-16">
						{#if modules && modules.modules.length > 0}
							{#each modules.modules as m}
								<section class="border-background-alt-2 grid grid-cols-4 border-t">
									<div class="col-span-full sm:col-span-1">
										<div class="border-foreground-alt-2 -mt-px inline-flex border-t pt-px">
											<div
												class="text-background-primary-alt-1 flex items-center justify-between gap-4 pt-6 font-semibold sm:pt-10"
											>
												<span>Module {pad2(m.prefix)}</span>
												{#if isAssetEditMode}
													{@const moduleAssets = getAllAssetsInModule(m.prefix)}
													{@const moduleSelectedCount = moduleAssets.filter(
														(a) => selectedAssets[a.id]
													).length}
													{@const moduleAllSelected =
														moduleSelectedCount === moduleAssets.length && moduleAssets.length > 0}
													{@const moduleIndeterminate =
														moduleSelectedCount > 0 && moduleSelectedCount < moduleAssets.length}
													<div class="sm:hidden">
														<Checkbox
															checked={moduleAllSelected}
															indeterminate={moduleIndeterminate}
															onclick={(e: MouseEvent) => {
																e.preventDefault();
																toggleModule(m.prefix);
															}}
														/>
													</div>
												{/if}
											</div>
										</div>
									</div>

									<div class="col-span-full pt-6 sm:col-span-3 sm:pt-10">
										<div class="max-w-2xl">
											<!-- Module title -->
											{#if m.module !== '(no chapter)'}
												<div class="relative text-2xl font-medium text-pretty">
													{#if isAssetEditMode}
														{@const moduleAssets = getAllAssetsInModule(m.prefix)}
														{@const moduleSelectedCount = moduleAssets.filter(
															(a) => selectedAssets[a.id]
														).length}
														{@const moduleAllSelected =
															moduleSelectedCount === moduleAssets.length &&
															moduleAssets.length > 0}
														{@const moduleIndeterminate =
															moduleSelectedCount > 0 && moduleSelectedCount < moduleAssets.length}
														<div class="absolute -top-0.5 -left-8 hidden sm:block">
															<Checkbox
																checked={moduleAllSelected}
																indeterminate={moduleIndeterminate}
																onclick={(e: MouseEvent) => {
																	e.preventDefault();
																	toggleModule(m.prefix);
																}}
															/>
														</div>
													{/if}
													<span>{m.module}</span>
												</div>
											{/if}

											<!-- Module details -->
											<ol class="mt-8 space-y-6 sm:mt-10">
												{#each m.lessons as lesson}
													{@const isCollection = lesson.assets.length > 1}
													{@const totalVideoDuration = lesson.totalVideoDuration}

													<li>
														<div class="flow-root">
															<div
																class="hover:bg-background-alt-2 -mx-3 -my-2 flex h-auto gap-2 px-3 py-2 text-sm whitespace-normal"
																class:items-center={!isAssetEditMode}
																class:items-start={isAssetEditMode}
																class:select-none={isAssetEditMode}
															>
																<!-- Lesson status or checkbox -->
																{#if isAssetEditMode}
																	{@const allSelected = lesson.assets.every(
																		(a: AssetModel) => selectedAssets[a.id]
																	)}
																	{@const someSelected = lesson.assets.some(
																		(a: AssetModel) => selectedAssets[a.id]
																	)}
																	{@const isOngoing = lesson.started && !lesson.completed}
																	{@const shouldBeIndeterminate =
																		isOngoing || (someSelected && !allSelected)}
																	<div class="mt-0.5 shrink-0">
																		<Checkbox
																			checked={allSelected && !isOngoing}
																			indeterminate={shouldBeIndeterminate}
																			onCheckedChange={() => {
																				toggleLessonAssets(lesson.assets);
																			}}
																		/>
																	</div>
																{:else if lesson.completed}
																	<TickCircleIcon
																		class="stroke-background-success fill-background-success [&_path]:stroke-foreground -mt-1.5 size-5 shrink-0 place-self-start stroke-1 [&_path]:stroke-1"
																	/>
																{:else if lesson.started}
																	<HalfCircleIcon
																		class="-mt-1.5 size-5 shrink-0 place-self-start fill-amber-700 text-amber-700"
																	/>
																{:else}
																	<PlayCircleIcon
																		class="stroke-foreground-alt-3 fill-background [&_polygon]:stroke-foreground-alt-2 [&_polygon]:fill-foreground-alt-2 -mt-1.5 size-5 shrink-0 place-self-start stroke-1"
																	/>
																{/if}

																{#if isAssetEditMode}
																	<div
																		class="flex flex-1 cursor-pointer flex-col gap-1.5 text-left"
																		role="button"
																		tabindex="0"
																		onclick={() => toggleLessonAssets(lesson.assets)}
																		onkeydown={(e: KeyboardEvent) => {
																			if (e.key === 'Enter' || e.key === ' ') {
																				e.preventDefault();
																				toggleLessonAssets(lesson.assets);
																			}
																		}}
																	>
																		<!-- Lesson title -->
																		<span class="text-foreground-alt-2 w-full font-semibold">
																			{lesson.prefix}. {lesson.title}
																		</span>

																		<!-- Lesson details -->
																		<div
																			class="relative flex w-full flex-col gap-0 text-sm select-none"
																		>
																			<div
																				class="flex w-full flex-row flex-wrap items-center gap-2"
																			>
																				<!-- Type -->
																				<span class="text-foreground-alt-3 whitespace-nowrap">
																					{#if isCollection}
																						collection
																					{:else}
																						{lesson.assets[0].type}
																					{/if}
																				</span>

																				<!-- Video duration -->
																				{#if totalVideoDuration > 0}
																					<DotIcon class="text-foreground-alt-3 text-xl" />
																					<span class="text-foreground-alt-3 whitespace-nowrap">
																						{prettyMs(totalVideoDuration * 1000)}
																					</span>
																				{/if}

																				<!-- Attachments -->
																				{#if lesson.attachments.length > 0 && !isAssetEditMode}
																					<DotIcon class="text-foreground-alt-3 text-xl" />
																					<Attachments
																						attachments={lesson.attachments}
																						courseId={course?.id ?? ''}
																						lessonId={lesson.id}
																					/>
																				{/if}
																			</div>
																		</div>
																	</div>
																{:else}
																	<Button
																		href={`/course/${course.id}/${lesson.id}`}
																		variant="ghost"
																		class="flex-1 hover:bg-transparent"
																		disabled={isScanning || course.maintenance || !course.available}
																		onclick={(e: MouseEvent) => {
																			if (isScanning || course?.maintenance || !course?.available) {
																				e.preventDefault();
																				e.stopPropagation();
																			}
																		}}
																	>
																		<div class="flex w-full flex-col gap-1.5">
																			<!-- Lesson title -->
																			<span class="text-foreground-alt-2 w-full font-semibold">
																				{lesson.prefix}. {lesson.title}
																			</span>

																			<!-- Lesson details -->
																			<div
																				class="relative flex w-full flex-col gap-0 text-sm select-none"
																			>
																				<div
																					class="flex w-full flex-row flex-wrap items-center gap-2"
																				>
																					<!-- Type -->
																					<span class="text-foreground-alt-3 whitespace-nowrap">
																						{#if isCollection}
																							collection
																						{:else}
																							{lesson.assets[0].type}
																						{/if}
																					</span>

																					<!-- Video duration -->
																					{#if totalVideoDuration > 0}
																						<DotIcon class="text-foreground-alt-3 text-xl" />
																						<span class="text-foreground-alt-3 whitespace-nowrap">
																							{prettyMs(totalVideoDuration * 1000)}
																						</span>
																					{/if}

																					<!-- Attachments -->
																					{#if lesson.attachments.length > 0}
																						<DotIcon class="text-foreground-alt-3 text-xl" />
																						<Attachments
																							attachments={lesson.attachments}
																							courseId={course?.id ?? ''}
																							lessonId={lesson.id}
																						/>
																					{/if}
																				</div>
																			</div>
																		</div>
																	</Button>
																{/if}
															</div>
														</div>
													</li>
												{/each}
											</ol>
										</div>
									</div>
								</section>
							{/each}
						{:else}
							<!-- Optional: loading/empty state -->
							<div class="text-foreground-alt-3 py-10 text-center">No modules to display.</div>
						{/if}
					</div>
				</div>
			</div>
		</div>

		<!-- Bottom sheet for asset management -->
		{#if isAssetEditMode}
			<div
				class="bg-background-alt-1 animate-in slide-in-from-bottom-4 border-background-alt-3 fixed bottom-4 left-1/2 z-50 flex -translate-x-1/2 items-center gap-6 rounded-md border px-4 py-2 shadow-lg duration-300"
			>
				<Button
					variant="outline"
					onclick={cancelAssetEditMode}
					disabled={isPostingAssets}
					class="h-auto px-6 py-1"
				>
					Cancel
				</Button>

				<Button
					variant="default"
					onclick={confirmAssetChanges}
					disabled={isPostingAssets}
					class="h-auto px-6 py-1"
				>
					{#if isPostingAssets}
						<Spinner class="bg-foreground-alt-1 size-2" />
					{:else}
						Confirm
					{/if}
				</Button>
			</div>
		{/if}
	{/if}
{:catch error}
	<div class="flex w-full flex-col items-center gap-2 pt-10">
		<WarningIcon class="text-foreground-error size-10" />
		<span class="text-lg">Failed to load page</span>
		<span class="text-foreground-alt-3 text-sm">{error.message}</span>
	</div>
{/await}
