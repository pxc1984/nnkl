<script lang="ts" module>
    import type {AppTypes} from "$app/types";
    import BookOpenIcon from "@lucide/svelte/icons/book-open";
    import NetworkIcon from "@lucide/svelte/icons/network";
    import Settings2Icon from "@lucide/svelte/icons/settings-2";
    import {Database, SearchIcon} from "@lucide/svelte";

    type Pathname = ReturnType<AppTypes["Pathname"]>;
    type NavUrl = "#" | Pathname | `http${string}`;
    type NavItem = {
        title: string;
        url: NavUrl;
    };
    type NavMainItem = NavItem & {
        // This should be `Component` after @lucide/svelte updates types
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        icon: any;
        isActive?: boolean;
        items?: NavItem[];
    };
    type ProjectItem = {
        name: string;
        preview: string;
        time: string;
        active?: boolean;
    };
    type SidebarUser = {
        name: string;
        email: string;
        avatar?: string | null;
    };
    type SidebarData = {
        user: SidebarUser;
        navMain: NavMainItem[];
        navSecondary: NavSecondaryItem[];
        queries: ProjectItem[];
    };
    type NavSecondaryItem = NavItem & {
        // This should be `Component` after @lucide/svelte updates types
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        icon: any;
    };

    export const defaultAppSidebarData: SidebarData = {
        user: {
            name: "Владимир Потанин",
            email: "potanin@nornickel.ru",
            avatar: "/potanin.jpg",
        },
        navMain: [
            {
                title: "Поиск",
                url: "/data/ask",
                icon: SearchIcon,
            },
            {
                title: "Карта знаний",
                url: "/data/graph",
                icon: NetworkIcon,
            },
            {
                title: "Материалы",
                url: "/data",
                icon: Database,
                isActive: true,
                items: [
                    {
                        title: "Список",
                        url: "/data",
                    },
                    {
                        title: "Загрузить",
                        url: "/data/upload",
                    },
                ],
            },
            {
                title: "Документация",
                url: "#",
                icon: BookOpenIcon,
                items: [
                    {
                        title: "Техническое задание",
                        url: "#",
                    },
                    {
                        title: "Быстрый старт",
                        url: "#",
                    },
                    {
                        title: "VCS",
                        url: "https://github.com/pxc1984/nnkl",
                    },
                ],
            },
            {
                title: "Настройки",
                url: "#",
                icon: Settings2Icon,
                items: [
                    {
                        title: "Общие",
                        url: "#",
                    },
                    {
                        title: "Доступы",
                        url: "#",
                    },
                ],
            },
        ],
        navSecondary: [
            // {
            // 	title: "Поддержка",
            // 	url: "#",
            // 	icon: LifeBuoyIcon,
            // },
            // {
            // 	title: "Обратная связь",
            // 	url: "#",
            // 	icon: SendIcon,
            // },
        ],
        queries: [
            // {
            // 	name: "Как подключить внутреннюю базу документов к поиску?",
            // 	preview: "Архитектура индексации, права доступа, обновление документов.",
            // 	time: "Сегодня",
            // 	active: true,
            // },
            // {
            // 	name: "Какие поля нужны для векторного индекса?",
            // 	preview: "Метаданные, чанки, source id, revision и owner.",
            // 	time: "Сегодня",
            // },
            // {
            // 	name: "Как ускорить поиск по PDF и DOCX?",
            // 	preview: "Предобработка, извлечение текста, кэширование и OCR.",
            // 	time: "Вчера",
            // },
            // {
            // 	name: "Как хранить версии документов?",
            // 	preview: "Immutable revisions, aliases и откат на предыдущую версию.",
            // 	time: "Вчера",
            // },
            // {
            // 	name: "Какие ограничения сделать для внешних источников?",
            // 	preview: "Rate limiting, allowlist, sanitization, audit trail.",
            // 	time: "2 дня назад",
            // },
            // {
            // 	name: "Как проектировать ответы со ссылками на источники?",
            // 	preview: "Цитаты, confidence score, deep links и превью документа.",
            // 	time: "3 дня назад",
            // },
            // {
            // 	name: "Как организовать обновление индекса по событию?",
            // 	preview: "Очередь задач, дедупликация, retry и фоновые воркеры.",
            // 	time: "На этой неделе",
            // },
            // {
            // 	name: "Что сохранять в истории запросов пользователей?",
            // 	preview: "Текст запроса, фильтры, выбранные документы и обратную связь.",
            // 	time: "На этой неделе",
            // },
        ],
    };

    export type {NavUrl};
    export type {SidebarData, SidebarUser};
</script>

<script lang="ts">
    import type {UserProfile} from "$lib/auth/types";
    import NavMain from "./nav-main.svelte";
    import NavProjects from "./nav-projects.svelte";
    import NavSecondary from "./nav-secondary.svelte";
    import NavUser from "./nav-user.svelte";
    import * as Sidebar from "$lib/components/ui/sidebar/index.js";
    import type {ComponentProps} from "svelte";

    let {
        ref = $bindable(null),
        currentUser = null,
        appSidebarData = defaultAppSidebarData,
        ...restProps
    }: ComponentProps<typeof Sidebar.Root> & {
        currentUser?: UserProfile | null;
        appSidebarData?: SidebarData;
    } = $props();

    const sidebarUser = $derived(
        currentUser
            ? {
                name: currentUser.name?.trim() || currentUser.email,
                email: currentUser.email,
                avatar: currentUser.avatarUrl || "/potanin.jpg",
            }
            : appSidebarData.user,
    );
</script>

<Sidebar.Root bind:ref variant="inset" {...restProps}>
    <Sidebar.Header>
        <Sidebar.Menu>
            <Sidebar.MenuItem>
                <Sidebar.MenuButton size="lg">
                    {#snippet child()}
                        <a href="https://nornickel.ru" class="flex items-center gap-2 font-medium">
                            <div class="z-logo header__logo header__logo--big z-logo--full-image"><img
                                    src="https://nornickel.ru/images/logo/logo-inverted-ru.svg" alt="logo"
                                    class="z-logo__img">
                            </div>
                        </a>
                    {/snippet}
                </Sidebar.MenuButton>
            </Sidebar.MenuItem>
        </Sidebar.Menu>
    </Sidebar.Header>
    <Sidebar.Content class="overflow-hidden">
        <NavMain items={appSidebarData.navMain}/>
        <NavProjects queries={appSidebarData.queries}/>
        <NavSecondary items={appSidebarData.navSecondary} class="mt-auto"/>
    </Sidebar.Content>
    <Sidebar.Footer>
        <NavUser user={sidebarUser}/>
    </Sidebar.Footer>
</Sidebar.Root>
