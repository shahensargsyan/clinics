# Clinics API — React + Ant Design Integration Guide

**Build tool:** Vite
**UI:** Ant Design v5 (Antd)
**Data fetching:** TanStack Query v5 (React Query)
**Codegen:** Orval (TypeScript types + React Query hooks)
**Auth:** Bearer Token (JWT) in `localStorage`
**Base URL:** `import.meta.env.VITE_API_BASE_URL`
**Spec source:** `api/openapi.yaml` — also served live at `GET /openapi.json`

> The backend is contract-first: `api/openapi.yaml` is the single source of truth. Every route requires a `bearerAuth` JWT except `POST /auth/login`. List endpoints return `{ data: T[], meta: { current_page, per_page, total, last_page } }`. Validation failures return `{ message: string, errors: { field: string[] } }` (HTTP 422). This guide wires all three into Antd + React Query cleanly.

---

## 1. Code Generation Setup (Orval)

### 1.1 Install

```bash
# runtime deps
npm install @tanstack/react-query axios

# dev-only codegen
npm install -D orval
```

### 1.2 `orval.config.ts` (production-ready)

Place at the repo root. It reads the spec, routes every request through a **custom Axios instance** (so auth/headers/errors are centralized), and emits split, tree-shakable React Query v5 hooks.

```ts
// orval.config.ts
import { defineConfig } from 'orval';

export default defineConfig({
  clinics: {
    input: {
      // Local spec for reproducible builds; swap for the live URL when verifying
      // against a deployed environment: 'https://<api-host>/openapi.json'
      target: './api/openapi.yaml',
    },
    output: {
      mode: 'tags-split',            // one folder per OpenAPI tag (patients/, auth/, …)
      target: './src/api/generated',
      schemas: './src/api/generated/model',
      client: 'react-query',
      httpClient: 'axios',
      clean: true,                   // wipe stale generated files each run
      prettier: true,
      override: {
        // Route EVERY generated call through our Axios wrapper (section 2.2).
        mutator: {
          path: './src/api/axios-instance.ts',
          name: 'customInstance',
        },
        query: {
          useQuery: true,
          useInfinite: false,
          // v5: smooth server-side pagination without flicker.
          options: {
            staleTime: 30_000,
          },
        },
      },
    },
    hooks: {
      afterAllFilesWrite: 'prettier --write',
    },
  },
});
```

### 1.3 npm script

```jsonc
// package.json
{
  "scripts": {
    "api:generate": "orval --config ./orval.config.ts"
  }
}
```

Run it whenever the spec changes:

```bash
npm run api:generate
```

Orval produces, per tag:

- **Hooks** — `useListPatients()`, `useCreatePatient()`, `useLoginUser()`, …
- **Query-key helpers** — `getListPatientsQueryKey(params)` (use these for cache invalidation; never hand-roll keys).
- **Types** — `Patient`, `PatientCreate`, `PaginatedPatients`, `PaginationMeta`, `LoginResponse`, …

---

## 2. Global App Architecture Setup

### 2.1 Environment

```bash
# .env.development
VITE_API_BASE_URL=http://localhost:8099

# .env.production
VITE_API_BASE_URL=https://<your-api-host>
```

### 2.2 The custom Axios instance (the Orval mutator)

Every generated hook calls `customInstance`. This is the one place auth headers are attached and `401` / `5xx` are handled globally.

```ts
// src/api/axios-instance.ts
import Axios, { AxiosError, AxiosRequestConfig } from 'axios';
import { message, notification } from 'antd';

export const AXIOS_INSTANCE = Axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
});

// --- Request: attach the bearer token (skip the public login route) ---
AXIOS_INSTANCE.interceptors.request.use((config) => {
  const token = localStorage.getItem('clinics_access_token');
  const isAuthCall = config.url?.includes('/auth/login');
  if (token && !isAuthCall) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// --- Response: centralize 401 + 5xx feedback ---
AXIOS_INSTANCE.interceptors.response.use(
  (response) => response,
  (error: AxiosError<{ message?: string }>) => {
    const status = error.response?.status;

    if (status === 401) {
      localStorage.removeItem('clinics_access_token');
      // Avoid a redirect loop if we're already on /login.
      if (!window.location.pathname.startsWith('/login')) {
        message.warning('Your session has expired. Please sign in again.');
        window.location.assign(`/login?returnUrl=${encodeURIComponent(window.location.pathname)}`);
      }
    } else if (status && status >= 500) {
      notification.error({
        message: 'Server error',
        description: error.response?.data?.message ?? 'Something went wrong. Please try again.',
      });
    }
    // 422 (validation) is intentionally NOT toasted here — it belongs on the
    // form (section 3.2). Re-throw so React Query's onError can handle it.
    return Promise.reject(error);
  },
);

// Orval calls this signature for every operation.
export const customInstance = <T>(
  config: AxiosRequestConfig,
  options?: AxiosRequestConfig,
): Promise<T> => {
  const source = Axios.CancelToken.source();
  const promise = AXIOS_INSTANCE({
    ...config,
    ...options,
    cancelToken: source.token,
  }).then(({ data }) => data);

  // Lets React Query cancel in-flight requests.
  // @ts-expect-error attach cancel for query cancellation
  promise.cancel = () => source.cancel('Query was cancelled by React Query');
  return promise;
};

export type ErrorType<E> = AxiosError<E>;
```

> **Antd v5 caveat.** `message`/`notification` imported statically work outside React (perfect for interceptors) but won't pick up `ConfigProvider` theme tokens. That's an acceptable trade-off here. For **in-component** feedback, use the context-aware `App.useApp()` hook (wired below) instead of the static import.

### 2.3 `main.tsx` — providers

```tsx
// src/main.tsx
import React from 'react';
import ReactDOM from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { App as AntApp, ConfigProvider, theme } from 'antd';
import App from './App';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,        // data considered fresh for 30s
      gcTime: 5 * 60_000,       // cache retained 5 min after unused
      retry: 1,                 // one retry; auth/validation errors shouldn't spin
      refetchOnWindowFocus: false,
    },
  },
});

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ConfigProvider
      theme={{ algorithm: theme.defaultAlgorithm, token: { colorPrimary: '#1677ff' } }}
    >
      {/* <AntApp> provides context-aware message/notification/modal to components */}
      <AntApp>
        <QueryClientProvider client={queryClient}>
          <App />
          <ReactQueryDevtools initialIsOpen={false} />
        </QueryClientProvider>
      </AntApp>
    </ConfigProvider>
  </React.StrictMode>,
);
```

---

## 3. Implementation Examples (Ant Design)

### Example 1 — Data Table (GET query) with server-side pagination

The backend paginates with `page` / `per_page` and returns totals in `meta`. Drive Antd's `<Table>` pagination from that, and use v5's `placeholderData: keepPreviousData` so the grid doesn't flicker between pages.

```tsx
// src/features/patients/PatientsTable.tsx
import { useState } from 'react';
import { Table, Typography } from 'antd';
import type { TablePaginationConfig, ColumnsType } from 'antd/es/table';
import { keepPreviousData } from '@tanstack/react-query';
import { useListPatients } from '../../api/generated/patients/patients';
import type { Patient } from '../../api/generated/model';

const columns: ColumnsType<Patient> = [
  { title: 'Name', dataIndex: 'full_name', key: 'full_name' },
  { title: 'Email', dataIndex: 'email', key: 'email', render: (v) => v ?? '—' },
  { title: 'Phone', dataIndex: 'phone', key: 'phone', render: (v) => v ?? '—' },
  {
    title: 'DOB',
    dataIndex: 'date_of_birth',
    key: 'date_of_birth',
    render: (v) => v ?? '—',
  },
];

export function PatientsTable() {
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(25);

  const { data, isLoading, isFetching } = useListPatients(
    { page, per_page: perPage, sort: '-created_at' },
    { query: { placeholderData: keepPreviousData } },
  );

  const pagination: TablePaginationConfig = {
    current: data?.meta.current_page ?? page,
    pageSize: data?.meta.per_page ?? perPage,
    total: data?.meta.total ?? 0,
    showSizeChanger: true,
    onChange: (nextPage, nextSize) => {
      setPage(nextPage);
      setPerPage(nextSize);
    },
  };

  return (
    <>
      <Typography.Title level={4}>Patients</Typography.Title>
      <Table<Patient>
        rowKey="id"
        columns={columns}
        dataSource={data?.data ?? []}
        loading={isLoading || isFetching}
        pagination={pagination}
      />
    </>
  );
}
```

> The hook's params are passed straight through to the query string and are part of the cache key — changing `page`/`per_page` refetches automatically. No manual `useEffect`.

### Example 2 — Form Submission (POST mutation) in a Modal

An Antd `<Form>` inside a `<Modal>` driven by `useCreatePatient()`. On success it **invalidates the table query** (grid auto-refreshes) and shows `message.success`. On a `422` it maps backend field errors onto the form.

```tsx
// src/features/patients/CreatePatientModal.tsx
import { Modal, Form, Input, Select, App } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import {
  useCreatePatient,
  getListPatientsQueryKey,
} from '../../api/generated/patients/patients';
import type { PatientCreate } from '../../api/generated/model';

interface Props {
  open: boolean;
  onClose: () => void;
}

type ValidationError = { message: string; errors: Record<string, string[]> };

export function CreatePatientModal({ open, onClose }: Props) {
  const [form] = Form.useForm<PatientCreate>();
  const { message } = App.useApp();          // context-aware (themed) feedback
  const queryClient = useQueryClient();

  const { mutate, isPending } = useCreatePatient({
    mutation: {
      onSuccess: () => {
        // Refresh every cached page of the patients list.
        queryClient.invalidateQueries({ queryKey: getListPatientsQueryKey() });
        message.success('Patient created');
        form.resetFields();
        onClose();
      },
      onError: (error) => {
        const res = (error as AxiosError<ValidationError>).response;
        if (res?.status === 422) {
          // Map { errors: { field: [msg] } } onto Antd form fields.
          form.setFields(
            Object.entries(res.data.errors).map(([name, errors]) => ({
              name: name as keyof PatientCreate,
              errors,
            })),
          );
        } else {
          message.error('Could not create patient');
        }
      },
    },
  });

  const handleOk = () => {
    form.validateFields().then((values) => mutate({ data: values }));
  };

  return (
    <Modal
      title="New patient"
      open={open}
      onOk={handleOk}
      onCancel={onClose}
      confirmLoading={isPending}
      destroyOnClose
    >
      <Form form={form} layout="vertical" requiredMark>
        <Form.Item
          label="First name"
          name="first_name"
          rules={[{ required: true, max: 100 }]}
        >
          <Input />
        </Form.Item>
        <Form.Item
          label="Last name"
          name="last_name"
          rules={[{ required: true, max: 100 }]}
        >
          <Input />
        </Form.Item>
        <Form.Item label="Email" name="email" rules={[{ type: 'email' }]}>
          <Input />
        </Form.Item>
        <Form.Item label="Gender" name="gender">
          <Select
            allowClear
            options={[
              { value: 'Male', label: 'Male' },
              { value: 'Female', label: 'Female' },
              { value: 'Other', label: 'Other' },
            ]}
          />
        </Form.Item>
        <Form.Item label="Phone" name="phone" rules={[{ max: 20 }]}>
          <Input />
        </Form.Item>
      </Form>
    </Modal>
  );
}
```

---

## 4. Architecture & State Best Practices

1. **Treat `src/api/generated/` as read-only build output.** Never hand-edit it — `clean: true` deletes edits on the next run. To add behavior, compose: wrap a generated hook in your own hook, or add logic in the Axios mutator. Regenerate (`npm run api:generate`) on every spec change and review the diff like a dependency bump.

2. **Invalidate with generated query keys, never string literals.** Always use `getListPatientsQueryKey()` (and its parameterized form) for `invalidateQueries`. Hard-coded keys silently rot when the spec changes; the generated helpers fail the build instead. After any create/update/delete mutation, invalidate the affected list and detail queries in `onSuccess`.

3. **Set caching deliberately, per data volatility.** Use the global `staleTime`/`gcTime` defaults for typical reference data; override per-query for fast-moving data (`staleTime: 0`) or near-static data (longer `staleTime`). For paginated tables, always pass `placeholderData: keepPreviousData` so Antd's `<Table>` keeps the previous page visible during fetch instead of flashing a spinner.

4. **Optimistic updates: pair the cache with Antd's UI state, and always provide rollback.** For instant-feel actions (toggles, inline edits), use `onMutate` to `cancelQueries`, snapshot via `getQueryData`, and `setQueryData` to the predicted value; on `onError`, restore the snapshot **and** surface `message.error`; in `onSettled`, `invalidateQueries` to reconcile with the server. Drive button/row spinners from the mutation's `isPending`, never from local ad-hoc booleans.

5. **Keep cross-cutting concerns out of components.** Auth header attachment, `401` redirects, and `5xx` notifications live in the Axios mutator (section 2.2) — exactly once. Components only handle domain-specific outcomes: `422` → map onto the Antd `<Form>`, success → `message.success` + cache invalidation. A component should never read `localStorage` for the token or build an `Authorization` header itself.

---

_Grounded in the actual Clinics spec: the `/auth/login` → `LoginResponse` flow, `bearerAuth` on every other route, the `{ data, meta }` pagination envelope consumed by the Antd `<Table>`, and the `{ message, errors }` 422 shape mapped onto the Antd `<Form>`._
