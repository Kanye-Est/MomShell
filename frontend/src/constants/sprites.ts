export interface SpriteData {
  id: string;
  src: string;
  left: string;
  top: string;
  width: string;
  rotate?: number;
  scaleX?: number;
  scaleY?: number;
  zIndex?: number;
  label?: string;
  labelSize?: string; // e.g. '1rem', '0.9rem'
  labelOffsetY?: string; // e.g. '-20%', '4px'
  /** Override position/size on mobile (max-width: 768px) */
  mobile?: {
    left?: string;
    top?: string;
    width?: string;
  };
  /** Override position/size on mobile landscape (max-height: 500px, orientation: landscape) */
  landscape?: {
    left?: string;
    top?: string;
    width?: string;
  };
}

import carImg from "@/assets/images/car.png";
import barImg from "@/assets/images/bar.png";
import stoneImg from "@/assets/images/stone.png";
import crabImg from "@/assets/images/crab.png";
import shellImg from "@/assets/images/shell.png";
import chairImg from "@/assets/images/chairs.png";
import mailboxImg from "@/assets/images/mailbox.png";

export const SPRITES: SpriteData[] = [
  {
    id: "car",
    src: carImg,
    left: "40%",
    top: "-2.5%",
    width: "50vw",
    label: "个人中心",
    labelSize: "1.2rem",
    labelOffsetY: "-7%",
    mobile: { width: "60vw", left: "38%", top: "26%" },
    landscape: { width: "40vw", left: "38%", top: "-8%" },
  },

  {
    id: "bar",
    src: barImg,
    left: "24%",
    top: "-15%",
    width: "60vw",
    label: "智育社区",
    labelSize: "1.2rem",
    labelOffsetY: "-9%",
    mobile: { width: "70vw", left: "21%", top: "22%" },
    landscape: { width: "45vw", left: "26%", top: "-15%" },
  },

  {
    id: "stone",
    src: stoneImg,
    left: "68%",
    top: "20%",
    width: "30vw",
    label: "智聊助手",
    labelSize: "1.2rem",
    labelOffsetY: "-18%",
    mobile: { width: "45vw", top: "26%", left: "71%" },
    landscape: { width: "25vw", left: "66%", top: "5%" },
  },

  {
    id: "crab",
    src: crabImg,
    left: "55%",
    top: "12%",
    width: "6vw",
    zIndex: 10,
    mobile: { width: "8vw", left: "49%", top: "18%" },
    landscape: { width: "6vw", left: "53%", top: "10%" },
  },

  {
    id: "shell",
    src: shellImg,
    left: "50.75%",
    top: "60%",
    width: "5vw",
    rotate: 0,
    zIndex: 10,
    label: "生成相片",
    labelSize: "1.2rem",
    mobile: { width: "12vw", left: "52%", top: "43%" },
    landscape: { width: "6vw", left: "48%", top: "53%" },
  },

  // Mom-only: blind box shells generated from Dad's completed tasks
  {
    id: "gift-shell",
    src: shellImg,
    left: "46%",
    top: "66%",
    width: "4.8vw",
    rotate: -18,
    zIndex: 10,
    label: "盲盒贝壳",
    labelSize: "1.1rem",
    mobile: { width: "12vw", left: "42%", top: "46%" },
    landscape: { width: "6vw", left: "42%", top: "58%" },
  },

  {
    id: "chair",
    src: chairImg,
    left: "56%",
    top: "2.5%",
    width: "40vw",
    rotate: 0,
    label: "同频任务",
    labelSize: "1.2rem",
    labelOffsetY: "-6%",
    mobile: { width: "40vw", left: "60%", top: "30%" },
    landscape: { width: "30vw", left: "56%", top: "-3%" },
  },

  {
    id: "mailbox",
    src: mailboxImg,
    left: "51.5%",
    top: "22%",
    width: "12vw",
    rotate: 0,
    label: "心愿签",
    labelSize: "1.2rem",
    mobile: { width: "15vw", left: "56%", top: "31%" },
    landscape: { width: "8vw", left: "50%", top: "15%" },
  },
];
