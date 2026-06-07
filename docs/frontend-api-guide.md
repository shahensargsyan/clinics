# Clinics API — Angular Integration Guide

**Project:** Clinics API
**Angular:** 19 (standalone APIs; patterns apply cleanly to 17+)
**Async style:** RxJS Observables (+ Signals for view state)
**Auth:** Bearer Token (JWT) persisted in `localStorage`
**Spec source:** `api/openapi.yaml` — also served live at `GET /openapi.json`
**Generator:** `@openapitools/openapi-generator-cli` (`typescript-angular`)

> The backend is contract-first: `api/openapi.yaml` is the single source of truth. The Go server emits `bearerAuth` (HTTP Bearer/JWT) on every route except `POST /auth/login`. Validation errors return Laravel's shape — `{ "message": string, "errors": { field: string[] } }` — which this guide handles centrally.

---

## 1. OpenAPI Generation Setup

### 1.1 Install the generator (pinned)

Pin the CLI and the generator version so every developer and CI run produces byte-identical output.

```bash
npm install -D @openapitools/openapi-generator-cli
```

Create **`openapitools.json`** at the repo root to lock the version:

```json
{
  "$schema": "node_modules/@openapitools/openapi-generator-cli/config.schema.json",
  "spaces": 2,
  "generator-cli": {
    "version": "7.10.0"
  }
}
```

### 1.2 Add the generation script

In **`package.json`**:

```json
{
  "scripts": {
    "api:generate": "openapi-generator-cli generate -c openapi-generator.config.yaml",
    "api:generate:live": "openapi-generator-cli generate -c openapi-generator.config.yaml -i https://<your-api-host>/openapi.json"
  }
}
```

Create **`openapi-generator.config.yaml`** at the repo root:

```yaml
inputSpec: ./api/openapi.yaml        # or the live /openapi.json URL
generatorName: typescript-angular
output: ./src/app/core/api           # generated code lives here — see 1.4
additionalProperties:
  ngVersion: "19.0.0"
  useSingleRequestParameter: true    # methods take ONE typed params object
  providedInRoot: true               # services are @Injectable({providedIn:'root'})
  fileNaming: kebab-case
  serviceSuffix: Service
  modelSuffix: ""
  enumPropertyNaming: original
  withInterceptorMixin: false        # we own auth via an HttpInterceptor (section 2)
  supportsES6: true
```

### 1.3 Generate

```bash
npm run api:generate
```

Because the Go API serves its own spec, you can also generate straight from a running instance — handy for verifying against a deployed environment:

```bash
npm run api:generate:live
```

### 1.4 Where the generated files live

Generate into a dedicated, clearly-named folder that signals "do not touch":

```
src/app/
├─ core/
│  ├─ api/                      # ← 100% GENERATED. Never hand-edit.
│  │  ├─ api/                   #   PatientsService, AppointmentsService, AuthService, …
│  │  ├─ model/                 #   Patient, PaginatedPatients, LoginResponse, …
│  │  ├─ configuration.ts       #   Configuration, BASE_PATH token
│  │  ├─ api.module.ts          #   ApiModule (used only by NgModule apps)
│  │  └─ index.ts               #   barrel export
│  ├─ interceptors/
│  │  └─ auth.interceptor.ts    # ← hand-written (section 2)
│  └─ auth/
│     └─ token.service.ts       # ← hand-written token storage helper
└─ features/
   └─ patients/                 # feature components consume the generated services
```

Treat `core/api/` as a build artifact (see section 4 for the commit-vs-CI decision).

---

## 2. Global Configuration & Authentication

### 2.1 Token storage helper

Keep all storage access in one place so it is trivial to swap `localStorage` for cookies later.

```ts
// src/app/core/auth/token.service.ts
import { Injectable } from '@angular/core';

const TOKEN_KEY = 'clinics_access_token';

@Injectable({ providedIn: 'root' })
export class TokenService {
  get token(): string | null {
    return localStorage.getItem(TOKEN_KEY);
  }
  set(token: string): void {
    localStorage.setItem(TOKEN_KEY, token);
  }
  clear(): void {
    localStorage.removeItem(TOKEN_KEY);
  }
  get isAuthenticated(): boolean {
    return !!this.token;
  }
}
```

> **Security note.** `localStorage` is readable by any script on the page, so a JWT stored there is exposed to XSS. It is the pragmatic choice for an SPA calling a cross-origin API and is what this guide assumes. If the API and frontend share a domain, prefer the backend issuing an **HTTP-only, Secure, SameSite cookie** instead — in that case you drop the interceptor below and set `withCredentials: true` on the `Configuration` (section 2.4).

### 2.2 The functional auth interceptor

Modern Angular uses functional interceptors registered with `withInterceptors`. This one attaches the bearer token to every outgoing request and centralizes the `401` response.

```ts
// src/app/core/interceptors/auth.interceptor.ts
import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';
import { TokenService } from '../auth/token.service';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const tokens = inject(TokenService);
  const router = inject(Router);

  // Never attach a token to the public login endpoint.
  const isAuthCall = req.url.includes('/auth/login');
  const token = tokens.token;

  const authedReq =
    token && !isAuthCall
      ? req.clone({ setHeaders: { Authorization: `Bearer ${token}` } })
      : req;

  return next(authedReq).pipe(
    catchError((err: HttpErrorResponse) => {
      // Token expired or invalid → drop it and bounce to login.
      if (err.status === 401 && !isAuthCall) {
        tokens.clear();
        router.navigate(['/login'], {
          queryParams: { returnUrl: router.url },
        });
      }
      return throwError(() => err);
    }),
  );
};
```

### 2.3 Register HttpClient, the interceptor, and the API base path (standalone)

```ts
// src/app/app.config.ts
import { ApplicationConfig } from '@angular/core';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideRouter } from '@angular/router';

import { authInterceptor } from './core/interceptors/auth.interceptor';
import { BASE_PATH } from './core/api';
import { environment } from '../environments/environment';
import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideRouter(routes),
    provideHttpClient(withInterceptors([authInterceptor])),

    // The generated services read BASE_PATH for the server URL.
    // Drive it from environment files — never hardcode.
    { provide: BASE_PATH, useValue: environment.apiBaseUrl },
  ],
};
```

```ts
// src/environments/environment.ts
export const environment = {
  production: false,
  apiBaseUrl: 'http://localhost:8099',
};

// src/environments/environment.prod.ts
export const environment = {
  production: true,
  apiBaseUrl: 'https://<your-api-host>',
};
```

> Because we generated with `providedInRoot: true`, the services are tree-shakable singletons — you do **not** import `ApiModule`. You only provide `BASE_PATH`.

### 2.4 NgModule variant (if you are not yet standalone)

```ts
// app.module.ts
import { ApiModule, Configuration } from './core/api';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { authInterceptor } from './core/interceptors/auth.interceptor';
import { environment } from '../environments/environment';

@NgModule({
  imports: [
    ApiModule.forRoot(
      () => new Configuration({ basePath: environment.apiBaseUrl }),
    ),
  ],
  providers: [
    provideHttpClient(withInterceptors([authInterceptor])),
  ],
})
export class AppModule {}
```

> **Do not** also set `Configuration.accessToken`. The generator can auto-attach the bearer for secured operations, but combining that with the interceptor double-sends the header. Pick **one** mechanism — this guide standardizes on the interceptor.

### 2.5 Logging in

```ts
// src/app/core/auth/auth.facade.ts
import { Injectable, inject } from '@angular/core';
import { map, tap } from 'rxjs';
import { AuthService } from '../api'; // generated
import { TokenService } from './token.service';

@Injectable({ providedIn: 'root' })
export class AuthFacade {
  private readonly api = inject(AuthService);
  private readonly tokens = inject(TokenService);

  /** username == email, matching the backend's Laravel guard. */
  login(username: string, password: string) {
    return this.api
      .loginUser({ loginRequest: { username, password } }) // single-param style
      .pipe(
        tap((res) => this.tokens.set(res.accessToken)),
        map((res) => res.user),
      );
  }

  logout() {
    this.tokens.clear();
  }
}
```

---

## 3. How to Use the Generated Services

A real-world list screen: paginated, searchable patients with explicit `loading` and `error` view-state, using **Signals** for state and `takeUntilDestroyed` for cleanup.

### 3.1 Service injection

Inject the generated service with `inject()` — no constructor boilerplate, no manual providers (it is `providedInRoot`).

```ts
private readonly patientsApi = inject(PatientsService);
```

### 3.2 Component implementation

```ts
// src/app/features/patients/patient-list.component.ts
import { Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { HttpErrorResponse } from '@angular/common/http';
import { finalize } from 'rxjs';

import { PatientsService, Patient, PaginationMeta } from '../../core/api';

@Component({
  selector: 'app-patient-list',
  standalone: true,
  templateUrl: './patient-list.component.html',
})
export class PatientListComponent {
  private readonly patientsApi = inject(PatientsService);
  private readonly destroyRef = inject(DestroyRef);

  // View state as signals — change-detection-friendly and zoneless-ready.
  readonly patients = signal<Patient[]>([]);
  readonly meta = signal<PaginationMeta | null>(null);
  readonly loading = signal(false);
  readonly error = signal<string | null>(null);

  // Query state.
  readonly page = signal(1);
  readonly search = signal('');

  ngOnInit(): void {
    this.load();
  }

  load(): void {
    this.loading.set(true);
    this.error.set(null);

    this.patientsApi
      .listPatients({
        page: this.page(),
        perPage: 25,
        search: this.search() || undefined,
        sort: '-created_at',
      })
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.loading.set(false)),
      )
      .subscribe({
        next: (res) => {
          this.patients.set(res.data);
          this.meta.set(res.meta);
        },
        error: (err: HttpErrorResponse) => {
          // Backend returns { message, errors? } — surface the message.
          this.error.set(err.error?.message ?? 'Failed to load patients.');
        },
      });
  }

  onSearch(term: string): void {
    this.search.set(term);
    this.page.set(1);
    this.load();
  }

  goToPage(p: number): void {
    this.page.set(p);
    this.load();
  }
}
```

**Alternative — the `async` pipe.** For read-only screens with no imperative refresh, expose an `Observable` and let the template subscribe/unsubscribe for you:

```ts
readonly patients$ = this.patientsApi
  .listPatients({ page: 1, perPage: 25 })
  .pipe(map((res) => res.data));
// template: @if (patients$ | async; as patients) { @for (p of patients; …) }
```

Use `takeUntilDestroyed` (shown above) whenever you `subscribe` imperatively; use the `async` pipe whenever the data flows straight to the view. Never leave a manual `subscribe` without teardown.

### 3.3 HTML template (Angular control flow)

```html
<!-- src/app/features/patients/patient-list.component.html -->
<section class="patients">
  <input
    type="search"
    placeholder="Search by name, email, phone…"
    (input)="onSearch($any($event.target).value)"
  />

  @if (loading()) {
    <div class="spinner" role="status" aria-live="polite">Loading patients…</div>
  } @else if (error()) {
    <div class="alert alert--error" role="alert">{{ error() }}</div>
  } @else {
    <table>
      <thead>
        <tr><th>Name</th><th>Email</th><th>Phone</th></tr>
      </thead>
      <tbody>
        @for (patient of patients(); track patient.id) {
          <tr>
            <td>{{ patient.fullName }}</td>
            <td>{{ patient.email ?? '—' }}</td>
            <td>{{ patient.phone ?? '—' }}</td>
          </tr>
        } @empty {
          <tr><td colspan="3">No patients found.</td></tr>
        }
      </tbody>
    </table>

    @if (meta(); as m) {
      <nav class="pager">
        <button [disabled]="m.currentPage <= 1" (click)="goToPage(m.currentPage - 1)">
          Previous
        </button>
        <span>Page {{ m.currentPage }} of {{ m.lastPage }} · {{ m.total }} total</span>
        <button [disabled]="m.currentPage >= m.lastPage" (click)="goToPage(m.currentPage + 1)">
          Next
        </button>
      </nav>
    }
  }
</section>
```

> Note how `patient.fullName`, `meta.currentPage`, and `meta.lastPage` are **typed** — the generator camelCases the spec's `full_name`, `current_page`, `last_page`. If the backend renames a field, the regenerated models break the build at compile time rather than failing silently at runtime. That is the whole point of contract-first.

---

## 4. Best Practices for the Team

1. **Never hand-edit anything under `src/app/core/api/`.** It is a build artifact. Manual edits are silently destroyed on the next `npm run api:generate`. If you need to adapt behavior, wrap the generated service in a hand-written **facade** (e.g. `AuthFacade` in 2.5) and put your logic there.

2. **Regenerate whenever `api/openapi.yaml` changes — and pin the generator version.** Treat a spec change like a dependency bump: run `npm run api:generate`, review the diff, and fix any resulting compile errors before merging. The pinned `generator-cli.version` in `openapitools.json` guarantees identical output across machines and CI.

3. **The OpenAPI spec is the contract — coordinate, don't improvise.** Do not work around a missing field by reaching into raw HTTP. If the API is wrong or incomplete, the spec gets fixed on the backend and re-published; the frontend regenerates. Frontend and backend must never drift.

4. **Decide once: commit the generated code, or generate in CI — not both ad hoc.** Recommended: **generate in CI** (add `npm run api:generate` as a pre-build step) and git-ignore `src/app/core/api/`. This keeps the spec as the only versioned source. If your CI cannot reach the spec at build time, commit the generated output instead and enforce "regenerated, not edited" in code review.

5. **Centralize cross-cutting concerns — auth and errors — outside generated code.** Bearer attachment, `401` handling, and the Laravel `{ message, errors }` error shape belong in the interceptor and shared utilities, never sprinkled into components. Components should only translate a typed response or a typed error into view-state (`loading` / `error` signals).

---

_Grounded in the actual Clinics spec: the `/auth/login` → `LoginResponse` flow, `bearerAuth` on every other route, the `{ data, meta }` pagination envelope, and the `{ message, errors }` validation-error shape._
