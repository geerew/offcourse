import {
	array,
	boolean,
	number,
	object,
	optional,
	picklist,
	string,
	type InferOutput
} from 'valibot';
import { BaseSchema } from './base-model';
import { BasePaginationSchema, type PaginationReqParams } from './pagination-model';

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const UserRoleSchema = picklist(['admin', 'user']);
export type UserRole = InferOutput<typeof UserRoleSchema>;

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

export const SelectUserRoles = [
	{ value: 'user', label: 'User' },
	{ value: 'admin', label: 'Admin' }
];

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// Course Progress
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Course progress schema
export const CourseProgressSchema = object({
	started: boolean(),
	startedAt: string(),
	percent: number(),
	completedAt: string()
});

export type CourseProgressModel = InferOutput<typeof CourseProgressSchema>;

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// Course
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Course schema
export const CourseSchema = object({
	...BaseSchema.entries,
	title: string(),
	path: optional(string()),
	hasCard: boolean(),
	cardHash: optional(string()),
	available: boolean(),
	duration: number(),
	initialScan: optional(boolean()),
	maintenance: boolean(),
	description: string(),
	scanStatus: optional(string()),
	progress: optional(CourseProgressSchema),
	favourited: optional(boolean())
});

export type CourseModel = InferOutput<typeof CourseSchema>;
export type CoursesModel = CourseModel[];

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Course create schema
export const CourseCreateSchema = object({
	title: string(),
	path: string()
});

export type CourseCreateModel = InferOutput<typeof CourseCreateSchema>;

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Paginated courses schema
export const CoursePaginationSchema = object({
	...BasePaginationSchema.entries,
	items: array(CourseSchema)
});

export type CoursePaginationModel = InferOutput<typeof CoursePaginationSchema>;

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Request parameters for courses
export type CourseReqParams = PaginationReqParams & {
	q?: string;
	withUserProgress?: boolean;
};

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// Course Tags
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Course tag create schema
export const CreateCourseTagSchema = object({
	tag: string()
});

export type CourseTagCreateModel = InferOutput<typeof CreateCourseTagSchema>;

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Course tag schema
export const CourseTagSchema = object({
	id: string(),
	tag: string()
});

export type CourseTagModel = InferOutput<typeof CourseTagSchema>;
export type CourseTagsModel = CourseTagModel[];
