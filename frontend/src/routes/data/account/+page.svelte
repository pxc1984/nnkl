<script lang="ts">
  import { browser } from "$app/environment";
  import { getApiErrorMessage } from "$lib/api/auth";
  import { authState, updateProfile } from "$lib/auth/store";
import * as Avatar from "$lib/components/ui/avatar/index.js";
import { Input } from "$lib/components/ui/input/index.js";
import { Button } from "$lib/components/ui/button/index.js";
import {
    FieldGroup,
    Field,
    FieldLabel,
    FieldDescription,
    FieldSeparator,
} from "$lib/components/ui/field/index.js";
import { CameraIcon, Trash2Icon, LoaderIcon } from "@lucide/svelte";

  let name = $state("");
  let avatarPreview = $state<string | null>(null);
  let selectedFile = $state<File | null>(null);
  let isSubmitting = $state(false);
  let errorMessage = $state("");
  let successMessage = $state("");

  // Pre-fill from current user on mount
  $effect(() => {
    if (!browser) return;
    const user = $authState.user;
    if (user) {
      name = user.name || "";
    }
  });

  const user = $derived($authState.user);
  const avatarUrl = $derived(
    avatarPreview ?? user?.avatarUrl ?? null,
  );
  const avatarFallback = $derived(
    user
      ? (user.name || "")
          .split(" ")
          .filter(Boolean)
          .slice(0, 2)
          .map((part) => part[0]?.toUpperCase() ?? "")
          .join("") || user.email.slice(0, 2).toUpperCase()
      : "??",
  );
  const hasChanges = $derived(
    name !== (user?.name ?? "") || selectedFile !== null,
  );

  function handleFileSelect(event: Event) {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    // Validate file type
    const allowed = [".jpg", ".jpeg", ".png", ".gif", ".webp"];
    const ext = "." + file.name.split(".").pop()?.toLowerCase();
    if (!allowed.includes(ext)) {
      errorMessage = "Неподдерживаемый формат. Используйте JPG, PNG, GIF или WebP.";
      return;
    }
    if (file.size > 5 * 1024 * 1024) {
      errorMessage = "Файл слишком большой. Максимальный размер — 5 МБ.";
      return;
    }

    errorMessage = "";
    selectedFile = file;
    avatarPreview = URL.createObjectURL(file);
  }

  function handleRemoveAvatar() {
    selectedFile = null;
    avatarPreview = null;
  }

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault();
    if (isSubmitting) return;

    errorMessage = "";
    successMessage = "";
    isSubmitting = true;

    try {
      await updateProfile(
        name !== (user?.name ?? "") ? name : undefined,
        selectedFile ?? undefined,
      );
      successMessage = "Профиль обновлён.";
      selectedFile = null;
      avatarPreview = null;
    } catch (error) {
      errorMessage = getApiErrorMessage(error, "Не удалось обновить профиль.");
    } finally {
      isSubmitting = false;
    }
  }
</script>

<div class="mx-auto w-full max-w-lg py-6">
  <form onsubmit={handleSubmit} class="flex flex-col gap-10">
    <div class="flex flex-col items-center gap-1 text-center">
      <h1 class="text-3xl font-semibold tracking-tight">Аккаунт</h1>
      <p class="text-muted-foreground text-sm">Управление профилем и фотографией</p>
    </div>

    {#if errorMessage}
      <div class="rounded-2xl border border-destructive/20 bg-destructive/10 px-4 py-3 text-sm text-destructive">
        {errorMessage}
      </div>
    {/if}
    {#if successMessage}
      <div class="rounded-2xl border-border/20 border bg-muted/20 px-4 py-3 text-sm text-foreground">
        {successMessage}
      </div>
    {/if}

    <FieldGroup>
      <!-- Avatar section -->
      <Field>
        <FieldLabel>Фотография профиля</FieldLabel>
        <div class="flex items-center gap-6">
          <div class="relative">
            <Avatar.Root class="size-24 rounded-full">
              {#if avatarUrl}
                <Avatar.Image src={avatarUrl} alt="Фото профиля" />
              {/if}
              <Avatar.Fallback class="rounded-full text-2xl">{avatarFallback}</Avatar.Fallback>
            </Avatar.Root>
            {#if selectedFile}
              <button
                type="button"
                onclick={handleRemoveAvatar}
                class="absolute -top-1 -right-1 flex size-6 items-center justify-center rounded-full bg-destructive text-destructive-foreground hover:bg-destructive/90"
              >
                <Trash2Icon class="size-3.5" />
              </button>
            {/if}
          </div>
          <div class="flex flex-col gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onclick={() => document.getElementById("avatar-input")?.click()}
            >
              <CameraIcon class="size-4" />
              {selectedFile ? "Изменить" : "Загрузить"}
            </Button>
            <input
              id="avatar-input"
              type="file"
              accept=".jpg,.jpeg,.png,.gif,.webp"
              class="hidden"
              onchange={handleFileSelect}
            />
            <FieldDescription>JPG, PNG, GIF или WebP. До 5 МБ.</FieldDescription>
          </div>
        </div>
      </Field>

      <FieldSeparator />

      <!-- Name field -->
      <Field>
        <FieldLabel for="name-input">Имя</FieldLabel>
        <Input id="name-input" type="text" placeholder="Ваше имя" bind:value={name} />
      </Field>
    </FieldGroup>

    <Button type="submit" disabled={!hasChanges || isSubmitting} class="self-start">
      {#if isSubmitting}
        <LoaderIcon class="size-4 animate-spin" />
        Сохраняем...
      {:else}
        Сохранить
      {/if}
    </Button>
  </form>
</div>