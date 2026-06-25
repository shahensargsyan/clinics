# Clinics Admin — React Feature Implementation Guide

**Build tool:** Vite · **UI:** Ant Design v5 · **Data:** TanStack Query v5 · **Codegen:** Orval
**Auth:** Bearer JWT + protected admin routing
**Spec source:** `api/openapi.yaml` (also served live at `GET /openapi.json`)

> Contract-first: `api/openapi.yaml` is the single source of truth. Every route needs a `bearerAuth` JWT except `POST /auth/login`. Lists return `{ data: T[], meta: { current_page, per_page, total, last_page } }`. Validation failures return `{ message, errors: { field: string[] } }` (HTTP 422).

### Generated hook reference (real names + module paths)

Orval splits output by OpenAPI **tag**. The names below are what Orval actually emits from this spec — use them verbatim.

| Feature | Hook | Module |
|---|---|---|
| Login | `useLoginUser` | `generated/auth/auth.ts` |
| Create patient | `useCreatePatient` | `generated/patients/patients.ts` |
| Get / update patient | `useGetPatient`, `useUpdatePatient` | `generated/patients/patients.ts` |
| Appointments | `useListAppointments`, `useCreateAppointment` | `generated/appointments/appointments.ts` |
| Medical history | `useListMedicalHistory`, `useCreateMedicalHistory` | `generated/medical-history/medical-history.ts` |
| Orthodontics history | `useListOrthodonticsMedicalHistories`, `useCreateOrthodonticsMedicalHistory` | `generated/clinical/clinical.ts` |
| Invoices | `useListInvoices` | `generated/invoices/invoices.ts` |
| Aggregate snapshot | `useGetPatientSummary` | `generated/clinical/clinical.ts` |

> Note the prompt's example names (`useAdminLogin`, `useGetPatientById`, `useGetAppointments`) map to `useLoginUser`, `useGetPatient`, `useListAppointments`. **Do not alias** — call the generated names so the build breaks if the spec changes.

---

## 1. Code Generation Setup (Orval)

```ts
// orval.config.ts
import { defineConfig } from 'orval';

export default defineConfig({
  clinics: {
    input: { target: './api/openapi.yaml' }, // or 'https://<api-host>/openapi.json'
    output: {
      mode: 'tags-split',
      target: './src/api/generated',
      schemas: './src/api/generated/model',
      client: 'react-query',
      httpClient: 'axios',
      clean: true,
      prettier: true,
      override: {
        mutator: { path: './src/api/axios-instance.ts', name: 'customInstance' },
        query: { useQuery: true, options: { staleTime: 30_000 } },
      },
    },
    hooks: { afterAllFilesWrite: 'prettier --write' },
  },
});
```

```jsonc
// package.json
{ "scripts": { "api:generate": "orval --config ./orval.config.ts" } }
```

Run `npm run api:generate` after every spec change; review the diff before committing.

---

## 2. Authentication & Admin Login Flow

### 2.1 Token storage (abstracted)

```ts
// src/api/token-storage.ts
const KEY = 'clinics_access_token';
export const tokenStorage = {
  get: () => localStorage.getItem(KEY),
  set: (t: string) => localStorage.setItem(KEY, t),
  clear: () => localStorage.removeItem(KEY),
};
```

> **Security note.** `localStorage` is XSS-readable. It is the pragmatic SPA choice for a cross-origin API. If the admin app shares a domain with the API, prefer a backend-issued **HTTP-only, Secure cookie** and drop the request interceptor below. Keeping all access behind `tokenStorage` makes that swap a one-file change.

### 2.2 Axios interceptor (the Orval mutator)

```ts
// src/api/axios-instance.ts
import Axios, { AxiosError, AxiosRequestConfig } from 'axios';
import { message } from 'antd';
import { tokenStorage } from './token-storage';

export const AXIOS_INSTANCE = Axios.create({ baseURL: import.meta.env.VITE_API_BASE_URL });

AXIOS_INSTANCE.interceptors.request.use((config) => {
  const token = tokenStorage.get();
  if (token && !config.url?.includes('/auth/login')) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

AXIOS_INSTANCE.interceptors.response.use(
  (r) => r,
  (error: AxiosError) => {
    if (error.response?.status === 401 && !window.location.pathname.startsWith('/login')) {
      tokenStorage.clear();
      message.warning('Session expired. Please sign in again.');
      window.location.assign(`/login?returnUrl=${encodeURIComponent(window.location.pathname)}`);
    }
    return Promise.reject(error); // 422 handled on forms; 5xx can be added here
  },
);

export const customInstance = <T>(config: AxiosRequestConfig, options?: AxiosRequestConfig): Promise<T> => {
  const source = Axios.CancelToken.source();
  const promise = AXIOS_INSTANCE({ ...config, ...options, cancelToken: source.token }).then(({ data }) => data);
  // @ts-expect-error attach cancel for React Query
  promise.cancel = () => source.cancel('cancelled');
  return promise;
};
```

### 2.3 Protected admin routing

```tsx
// src/auth/RequireAuth.tsx
import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { tokenStorage } from '../api/token-storage';

export function RequireAuth() {
  const location = useLocation();
  if (!tokenStorage.get()) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }
  return <Outlet />;
}
```

```tsx
// src/router.tsx (excerpt)
import { createBrowserRouter } from 'react-router-dom';
import { RequireAuth } from './auth/RequireAuth';

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    element: <RequireAuth />,           // everything below requires a token
    children: [
      { path: '/patients', element: <PatientsListPage /> },
      { path: '/patients/new', element: <PatientCreatePage /> },
      { path: '/patients/:id', element: <PatientEditPage /> },
    ],
  },
]);
```

### 2.4 Admin login page

```tsx
// src/features/auth/LoginPage.tsx
import { Form, Input, Button, Card, App } from 'antd';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useLoginUser } from '../../api/generated/auth/auth';
import type { LoginRequest } from '../../api/generated/model';
import { tokenStorage } from '../../api/token-storage';

export function LoginPage() {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const [params] = useSearchParams();

  const { mutate, isPending } = useLoginUser({
    mutation: {
      onSuccess: (res) => {
        tokenStorage.set(res.access_token);
        message.success(`Welcome, ${res.user.name}`);
        navigate(params.get('returnUrl') ?? '/patients', { replace: true });
      },
      onError: () => message.error('Invalid email or password'),
    },
  });

  // The backend names the email field `username` (Laravel guard parity).
  const onFinish = (values: LoginRequest) => mutate({ data: values });

  return (
    <Card title="Admin sign in" style={{ maxWidth: 360, margin: '10vh auto' }}>
      <Form layout="vertical" onFinish={onFinish} disabled={isPending}>
        <Form.Item label="Email" name="username" rules={[{ required: true, type: 'email' }]}>
          <Input autoComplete="username" />
        </Form.Item>
        <Form.Item label="Password" name="password" rules={[{ required: true }]}>
          <Input.Password autoComplete="current-password" />
        </Form.Item>
        <Button type="primary" htmlType="submit" block loading={isPending}>
          Sign in
        </Button>
      </Form>
    </Card>
  );
}
```

---

## 3. Patient Creation Page

Antd v5 `<DatePicker>` works in `dayjs` objects; the API wants `date_of_birth` as `YYYY-MM-DD`. Convert on submit.

```tsx
// src/features/patients/PatientCreatePage.tsx
import { Form, Input, Select, DatePicker, Button, Card, App } from 'antd';
import { useNavigate } from 'react-router-dom';
import { AxiosError } from 'axios';
import type { Dayjs } from 'dayjs';
import { useCreatePatient } from '../../api/generated/patients/patients';
import type { PatientCreate } from '../../api/generated/model';

type FormShape = Omit<PatientCreate, 'date_of_birth'> & { date_of_birth?: Dayjs };
type ValidationError = { message: string; errors: Record<string, string[]> };

export function PatientCreatePage() {
  const [form] = Form.useForm<FormShape>();
  const { message } = App.useApp();
  const navigate = useNavigate();

  const { mutate, isPending } = useCreatePatient({
    mutation: {
      onSuccess: (res) => {
        message.success('Patient created');
        navigate(`/patients/${res.data.id}`); // jump straight to the edit dashboard
      },
      onError: (error) => {
        const r = (error as AxiosError<ValidationError>).response;
        if (r?.status === 422) {
          form.setFields(Object.entries(r.data.errors).map(([name, errors]) => ({ name, errors })));
        } else {
          message.error('Could not create patient');
        }
      },
    },
  });

  const onFinish = (values: FormShape) =>
    mutate({
      data: { ...values, date_of_birth: values.date_of_birth?.format('YYYY-MM-DD') },
    });

  return (
    <Card title="New patient" style={{ maxWidth: 640, margin: '24px auto' }}>
      <Form form={form} layout="vertical" onFinish={onFinish} disabled={isPending}>
        <Form.Item label="First name" name="first_name" rules={[{ required: true, max: 100 }]}>
          <Input />
        </Form.Item>
        <Form.Item label="Last name" name="last_name" rules={[{ required: true, max: 100 }]}>
          <Input />
        </Form.Item>
        <Form.Item label="Date of birth" name="date_of_birth">
          <DatePicker style={{ width: '100%' }} format="YYYY-MM-DD" />
        </Form.Item>
        <Form.Item label="Gender" name="gender">
          <Select
            allowClear
            options={['Male', 'Female', 'Other'].map((g) => ({ value: g, label: g }))}
          />
        </Form.Item>
        <Form.Item label="Email" name="email" rules={[{ type: 'email', max: 100 }]}>
          <Input />
        </Form.Item>
        <Form.Item label="Phone" name="phone" rules={[{ max: 20 }]}>
          <Input />
        </Form.Item>
        <Form.Item label="Address" name="address" rules={[{ max: 255 }]}>
          <Input.TextArea rows={2} />
        </Form.Item>
        <Button type="primary" htmlType="submit" loading={isPending}>
          Create patient
        </Button>
      </Form>
    </Card>
  );
}
```

---

## 4. Patient Edit & Management Dashboard (Tabbed)

The shell loads the patient header once and renders five tabs. Use `destroyInactiveTabPane` so each tab's queries fire **only when opened** — this keeps the dashboard light.

```tsx
// src/features/patients/PatientEditPage.tsx
import { useParams } from 'react-router-dom';
import { Tabs, Spin, Typography } from 'antd';
import { useGetPatient } from '../../api/generated/patients/patients';
import { ProfileTab } from './tabs/ProfileTab';
import { AppointmentsTab } from './tabs/AppointmentsTab';
import { MedicalHistoryTab } from './tabs/MedicalHistoryTab';
import { OrthodonticsTab } from './tabs/OrthodonticsTab';
import { InvoicesTab } from './tabs/InvoicesTab';

export function PatientEditPage() {
  const id = Number(useParams().id);
  const { data, isLoading } = useGetPatient(id);

  if (isLoading) return <Spin style={{ margin: '20vh auto', display: 'block' }} />;

  return (
    <div style={{ padding: 24 }}>
      <Typography.Title level={3}>{data?.data.full_name}</Typography.Title>
      <Tabs
        defaultActiveKey="profile"
        destroyInactiveTabPane
        items={[
          { key: 'profile', label: 'Profile', children: <ProfileTab patientId={id} /> },
          { key: 'appointments', label: 'Appointments', children: <AppointmentsTab patientId={id} /> },
          { key: 'medical', label: 'Medical History', children: <MedicalHistoryTab patientId={id} /> },
          { key: 'ortho', label: 'Orthodontics', children: <OrthodonticsTab patientId={id} /> },
          { key: 'invoices', label: 'Invoices', children: <InvoicesTab patientId={id} /> },
        ]}
      />
    </div>
  );
}
```

### Tab 1 — Patient Profile (read + update)

```tsx
// src/features/patients/tabs/ProfileTab.tsx
import { useEffect } from 'react';
import { Form, Input, Select, DatePicker, Button, App } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import dayjs, { Dayjs } from 'dayjs';
import {
  useGetPatient,
  useUpdatePatient,
  getGetPatientQueryKey,
} from '../../../api/generated/patients/patients';

export function ProfileTab({ patientId }: { patientId: number }) {
  const [form] = Form.useForm();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { data } = useGetPatient(patientId);

  // Pre-populate the form once data arrives (DatePicker needs a dayjs value).
  useEffect(() => {
    if (data?.data) {
      form.setFieldsValue({
        ...data.data,
        date_of_birth: data.data.date_of_birth ? dayjs(data.data.date_of_birth) : undefined,
      });
    }
  }, [data, form]);

  const { mutate, isPending } = useUpdatePatient({
    mutation: {
      onSuccess: () => {
        message.success('Profile saved');
        queryClient.invalidateQueries({ queryKey: getGetPatientQueryKey(patientId) });
      },
      onError: () => message.error('Save failed'),
    },
  });

  const onFinish = (values: { date_of_birth?: Dayjs; [k: string]: unknown }) =>
    mutate({
      id: patientId,
      data: { ...values, date_of_birth: values.date_of_birth?.format('YYYY-MM-DD') },
    });

  return (
    <Form form={form} layout="vertical" onFinish={onFinish} style={{ maxWidth: 560 }}>
      <Form.Item label="First name" name="first_name" rules={[{ required: true }]}>
        <Input />
      </Form.Item>
      <Form.Item label="Last name" name="last_name" rules={[{ required: true }]}>
        <Input />
      </Form.Item>
      <Form.Item label="Date of birth" name="date_of_birth">
        <DatePicker style={{ width: '100%' }} format="YYYY-MM-DD" />
      </Form.Item>
      <Form.Item label="Gender" name="gender">
        <Select allowClear options={['Male', 'Female', 'Other'].map((g) => ({ value: g, label: g }))} />
      </Form.Item>
      <Form.Item label="Email" name="email" rules={[{ type: 'email' }]}>
        <Input />
      </Form.Item>
      <Form.Item label="Phone" name="phone">
        <Input />
      </Form.Item>
      <Button type="primary" htmlType="submit" loading={isPending}>
        Save changes
      </Button>
    </Form>
  );
}
```

### Tab 2 — Appointments (table + create modal)

```tsx
// src/features/patients/tabs/AppointmentsTab.tsx
import { useState } from 'react';
import { Table, Button, Modal, Form, DatePicker, InputNumber, Input, Select, Tag, App, Space } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useQueryClient } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import dayjs from 'dayjs';
import {
  useListAppointments,
  useCreateAppointment,
  getListAppointmentsQueryKey,
} from '../../../api/generated/appointments/appointments';
import type { Appointment } from '../../../api/generated/model';

const STATUS_COLOR: Record<string, string> = {
  Scheduled: 'blue', Confirmed: 'cyan', Completed: 'green', Cancelled: 'red',
};

export function AppointmentsTab({ patientId }: { patientId: number }) {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm();

  const listKey = getListAppointmentsQueryKey({ patient_id: patientId });
  const { data, isLoading } = useListAppointments({ patient_id: patientId, sort: '-appointment_date_time' });

  const { mutate, isPending } = useCreateAppointment({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: listKey }); // table refreshes instantly
        message.success('Appointment created');
        setOpen(false);
        form.resetFields();
      },
      onError: (error) => {
        const r = (error as AxiosError<{ errors: Record<string, string[]> }>).response;
        if (r?.status === 422) {
          form.setFields(Object.entries(r.data.errors).map(([name, errors]) => ({ name, errors })));
        } else {
          message.error('Could not create appointment');
        }
      },
    },
  });

  const columns: ColumnsType<Appointment> = [
    {
      title: 'When',
      dataIndex: 'appointment_date_time',
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm'),
    },
    { title: 'Duration', dataIndex: 'duration_minutes', render: (v) => `${v} min` },
    { title: 'Reason', dataIndex: 'reason_for_visit', render: (v) => v ?? '—' },
    { title: 'Status', dataIndex: 'status', render: (s: string) => <Tag color={STATUS_COLOR[s]}>{s}</Tag> },
  ];

  const onCreate = () =>
    form.validateFields().then((v) =>
      mutate({
        data: {
          patient_id: patientId,
          user_id: v.user_id,
          appointment_date_time: v.appointment_date_time.toISOString(),
          duration_minutes: v.duration_minutes ?? 30,
          reason_for_visit: v.reason_for_visit,
          status: v.status ?? 'Scheduled',
        },
      }),
    );

  return (
    <>
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" onClick={() => setOpen(true)}>New appointment</Button>
      </Space>
      <Table<Appointment> rowKey="id" loading={isLoading} columns={columns} dataSource={data?.data ?? []} />

      <Modal title="New appointment" open={open} onOk={onCreate} confirmLoading={isPending}
        onCancel={() => setOpen(false)} destroyOnClose>
        <Form form={form} layout="vertical">
          <Form.Item label="Clinician (user id)" name="user_id" rules={[{ required: true }]}>
            <InputNumber style={{ width: '100%' }} min={1} />
          </Form.Item>
          <Form.Item label="Date & time" name="appointment_date_time" rules={[{ required: true }]}>
            <DatePicker showTime style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="Duration (min)" name="duration_minutes" initialValue={30}>
            <InputNumber style={{ width: '100%' }} min={1} max={600} />
          </Form.Item>
          <Form.Item label="Reason" name="reason_for_visit">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item label="Status" name="status" initialValue="Scheduled">
            <Select options={Object.keys(STATUS_COLOR).map((s) => ({ value: s, label: s }))} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
```

### Tab 3 — Medical History

```tsx
// src/features/patients/tabs/MedicalHistoryTab.tsx
import { Table, Button, Modal, Form, Input, App, Space } from 'antd';
import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import {
  useListMedicalHistory,
  useCreateMedicalHistory,
  getListMedicalHistoryQueryKey,
} from '../../../api/generated/medical-history/medical-history';
import type { MedicalHistory } from '../../../api/generated/model';

export function MedicalHistoryTab({ patientId }: { patientId: number }) {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm();

  const key = getListMedicalHistoryQueryKey({ patient_id: patientId });
  const { data, isLoading } = useListMedicalHistory({ patient_id: patientId });

  const { mutate, isPending } = useCreateMedicalHistory({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: key });
        message.success('Record added');
        setOpen(false);
        form.resetFields();
      },
    },
  });

  return (
    <>
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" onClick={() => setOpen(true)}>Add record</Button>
      </Space>
      <Table<MedicalHistory>
        rowKey="id"
        loading={isLoading}
        dataSource={data?.data ?? []}
        columns={[
          { title: 'Condition', dataIndex: 'condition_name' },
          { title: 'Notes', dataIndex: 'notes', render: (v) => v ?? '—' },
        ]}
      />
      <Modal title="New medical record" open={open} confirmLoading={isPending} destroyOnClose
        onCancel={() => setOpen(false)}
        onOk={() => form.validateFields().then((v) => mutate({ data: { patient_id: patientId, ...v } }))}>
        <Form form={form} layout="vertical">
          <Form.Item label="Condition" name="condition_name" rules={[{ required: true, max: 255 }]}>
            <Input />
          </Form.Item>
          <Form.Item label="Notes" name="notes">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
```

### Tab 4 — Orthodontics Medical History

This record is a single longitudinal clinical sheet per patient (lives under the `Clinical` tag in the generated client). Treat the latest entry as the active one; "Save" creates a new versioned record.

```tsx
// src/features/patients/tabs/OrthodonticsTab.tsx
import { Form, Input, Button, App, Empty } from 'antd';
import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import {
  useListOrthodonticsMedicalHistories,
  useCreateOrthodonticsMedicalHistory,
  getListOrthodonticsMedicalHistoriesQueryKey,
} from '../../../api/generated/clinical/clinical';

const FIELDS = [
  ['main_complaints', 'Main complaints'],
  ['functional_disturbances', 'Functional disturbances'],
  ['ent_pathology', 'ENT pathology'],
  ['postural_disturbances', 'Postural disturbances'],
  ['biometric_findings', 'Biometric findings'],
  ['treatment_plan', 'Treatment plan'],
] as const;

export function OrthodonticsTab({ patientId }: { patientId: number }) {
  const [form] = Form.useForm();
  const { message } = App.useApp();
  const queryClient = useQueryClient();

  const key = getListOrthodonticsMedicalHistoriesQueryKey({ patient_id: patientId });
  const { data, isLoading } = useListOrthodonticsMedicalHistories({ patient_id: patientId, sort: '-created_at' });
  const latest = data?.data?.[0];

  useEffect(() => { if (latest) form.setFieldsValue(latest); }, [latest, form]);

  const { mutate, isPending } = useCreateOrthodonticsMedicalHistory({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: key });
        message.success('Orthodontic history saved');
      },
    },
  });

  if (isLoading) return null;

  return (
    <Form form={form} layout="vertical" style={{ maxWidth: 640 }}
      onFinish={(v) => mutate({ data: { patient_id: patientId, ...v } })}>
      {!latest && <Empty description="No orthodontic history yet — fill in below to start one." />}
      {FIELDS.map(([name, label]) => (
        <Form.Item key={name} label={label} name={name}>
          <Input.TextArea rows={2} />
        </Form.Item>
      ))}
      <Button type="primary" htmlType="submit" loading={isPending}>Save</Button>
    </Form>
  );
}
```

### Tab 5 — Invoices (read-only)

```tsx
// src/features/patients/tabs/InvoicesTab.tsx
import { Table, Tag, Alert } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useListInvoices } from '../../../api/generated/invoices/invoices';
import { useListAppointments } from '../../../api/generated/appointments/appointments';
import type { Invoice } from '../../../api/generated/model';

// Backend enum is Pending | Paid | Partial. "Overdue" is NOT a stored status —
// derive it client-side from invoice_date when still unpaid.
function statusTag(inv: Invoice) {
  if (inv.payment_status === 'Paid') return <Tag color="green">Paid</Tag>;
  if (inv.payment_status === 'Partial') return <Tag color="gold">Partial</Tag>;
  const overdue = dayjs(inv.invoice_date).isBefore(dayjs(), 'day');
  return <Tag color={overdue ? 'red' : 'default'}>{overdue ? 'Overdue' : 'Pending'}</Tag>;
}

export function InvoicesTab({ patientId }: { patientId: number }) {
  // Invoices are keyed by appointment_id (1:1), not patient_id. Resolve the
  // patient's appointments first, then pull invoices per appointment.
  const { data: appts } = useListAppointments({ patient_id: patientId, per_page: 100 });
  const apptIds = new Set(appts?.data?.map((a) => a.id));

  const { data, isLoading } = useListInvoices({ per_page: 100, sort: '-invoice_date' });
  const rows = (data?.data ?? []).filter((inv) => apptIds.has(inv.appointment_id));

  const columns: ColumnsType<Invoice> = [
    { title: 'Date', dataIndex: 'invoice_date' },
    { title: 'Total', dataIndex: 'total_amount', align: 'right' },
    { title: 'Paid', dataIndex: 'amount_paid', align: 'right' },
    { title: 'Balance', dataIndex: 'remaining_balance', align: 'right' },
    { title: 'Status', key: 'status', render: (_, inv) => statusTag(inv) },
  ];

  return (
    <>
      <Alert
        type="info"
        showMessage={false}
        message="Invoices are linked to appointments. A dedicated /patients/{id}/invoices endpoint would remove the client-side join below."
        style={{ marginBottom: 16 }}
      />
      <Table<Invoice> rowKey="id" loading={isLoading} columns={columns} dataSource={rows} pagination={false} />
    </>
  );
}
```

> **Architect's note (backend gap).** The spec exposes invoices filtered by `appointment_id`, not `patient_id`, so the tab does a client-side join through appointments. For production, request a backend `patient_id` filter on `GET /invoices` (or a `GET /patients/{id}/invoices` route). Until then, the `per_page: 100` cap above is a pragmatic ceiling, not a real solution.

---

## 5. Cache Invalidation & UI Best Practices

1. **Invalidate with generated query-key helpers — never string literals.** After every mutation, call `queryClient.invalidateQueries({ queryKey: getList…QueryKey(params) })` for the affected list and `getGet…QueryKey(id)` for the affected detail. Example: saving the Profile tab invalidates `getGetPatientQueryKey(id)` so the dashboard header (`full_name`) updates immediately. Hand-rolled keys silently rot when the spec changes; the generated helpers fail the build instead.

2. **Match invalidation params to query params.** A list hook called with `{ patient_id }` produces a key that *includes* `{ patient_id }`. Invalidate with the **same** params object, or use a partial key (`getListAppointmentsQueryKey()` without args) to invalidate every variation. Mismatched params are the #1 cause of "I created a record but the table didn't refresh."

3. **Scope queries to the open tab with `destroyInactiveTabPane`.** Each tab owns its queries; destroying inactive panes means a tab's data is fetched fresh when opened and never serves stale cache from a previous visit. For cross-tab consistency (e.g., creating an appointment that should affect the Invoices tab later), invalidate the *other* tab's key in `onSuccess` even though it isn't mounted — React Query marks it stale for the next mount.

4. **Optimistic updates pair the cache with Antd's UI state, always with rollback.** For inline status toggles, use `onMutate` to `cancelQueries` + snapshot via `getQueryData` + `setQueryData`; on `onError`, restore the snapshot **and** `message.error`; on `onSettled`, `invalidateQueries` to reconcile. Drive `<Button loading>`, `<Modal confirmLoading>`, and `<Table loading>` from the hook's `isPending`/`isFetching` — never from ad-hoc local booleans.

5. **Keep cross-cutting concerns in the mutator, domain outcomes in components.** Token attachment and `401` redirects live once in `axios-instance.ts`. Components only handle domain results: `422` → `form.setFields()` onto the Antd `<Form>`; success → `message.success` + targeted invalidation. A component must never read the token or build an `Authorization` header itself.

---

_Grounded in the actual Clinics spec: real operationIds and Orval module paths, the `Scheduled/Confirmed/Completed/Cancelled` appointment enum, the `Pending/Paid/Partial` invoice enum (with `Overdue` derived client-side), and the appointment-keyed invoice relationship._
