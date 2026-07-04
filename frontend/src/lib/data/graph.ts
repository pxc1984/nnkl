export type GraphNodeType =
  | "Material"
  | "Process"
  | "Equipment"
  | "Property"
  | "Experiment"
  | "Publication"
  | "Expert"
  | "Facility"
  | "Unknown";

export interface GraphNode {
  id: string;
  label: string;
  type: GraphNodeType;
}

export interface GraphEdge {
  source: string;
  target: string;
  label: string;
}

export interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export const NODE_COLORS: Record<GraphNodeType, string> = {
  Material: "#f97316",
  Process: "#3b82f6",
  Equipment: "#22c55e",
  Property: "#a855f7",
  Experiment: "#ef4444",
  Publication: "#eab308",
  Expert: "#ec4899",
  Facility: "#14b8a6",
  Unknown: "#6b7280",
};

export const mockGraph: GraphData = {
  nodes: [
    { id: "никель", label: "Никель", type: "Material" },
    { id: "электроэкстракция", label: "Электроэкстракция", type: "Process" },
    {
      id: "ванна_электроэкстракции",
      label: "Ванна электроэкстракции",
      type: "Equipment",
    },
    { id: "католит", label: "Католит", type: "Material" },
    {
      id: "циркуляция_электролита",
      label: "Циркуляция электролита",
      type: "Process",
    },
    { id: "Au", label: "Золото (Au)", type: "Material" },
    { id: "Ag", label: "Серебро (Ag)", type: "Material" },
    { id: "МПГ", label: "МПГ", type: "Material" },
    { id: "штейн", label: "Штейн", type: "Material" },
    { id: "шлак", label: "Шлак", type: "Material" },
    { id: "эксперт_1", label: "Иванов И.И.", type: "Expert" },
    { id: "эксперт_2", label: "Петров А.В.", type: "Expert" },
    {
      id: "публикация_1",
      label: "Обзор методов обессоливания",
      type: "Publication",
    },
  ],
  edges: [
    { source: "электроэкстракция", target: "никель", label: "применяется_для" },
    {
      source: "электроэкстракция",
      target: "ванна_электроэкстракции",
      label: "использует_оборудование",
    },
    {
      source: "циркуляция_электролита",
      target: "католит",
      label: "использует_материал",
    },
    {
      source: "циркуляция_электролита",
      target: "ванна_электроэкстракции",
      label: "происходит_в",
    },
    { source: "Au", target: "штейн", label: "распределяется_в" },
    { source: "Ag", target: "штейн", label: "распределяется_в" },
    { source: "МПГ", target: "шлак", label: "распределяется_в" },
    { source: "эксперт_1", target: "электроэкстракция", label: "эксперт_в" },
    {
      source: "эксперт_2",
      target: "циркуляция_электролита",
      label: "эксперт_в",
    },
    { source: "публикация_1", target: "электроэкстракция", label: "описывает" },
  ],
};
