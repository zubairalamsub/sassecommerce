'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export interface Review {
  id: string;
  productId: string;
  userId: string;
  userName: string;
  rating: number;
  title: string;
  comment: string;
  createdAt: string;
}

interface ReviewState {
  reviews: Review[];
  addReview: (review: Omit<Review, 'id' | 'createdAt'>) => void;
  getProductReviews: (productId: string) => Review[];
  getAverageRating: (productId: string) => { average: number; count: number };
}

export const useReviewStore = create<ReviewState>()(
  persist(
    (set, get) => ({
      reviews: [],

      addReview: (data) => {
        const review: Review = {
          ...data,
          id: `rev-${Date.now()}`,
          createdAt: new Date().toISOString(),
        };
        set((state) => ({ reviews: [review, ...state.reviews] }));
      },

      getProductReviews: (productId) =>
        get().reviews.filter((r) => r.productId === productId),

      getAverageRating: (productId) => {
        const productReviews = get().reviews.filter((r) => r.productId === productId);
        if (productReviews.length === 0) return { average: 0, count: 0 };
        const sum = productReviews.reduce((s, r) => s + r.rating, 0);
        return { average: sum / productReviews.length, count: productReviews.length };
      },
    }),
    { name: 'reviews-storage' },
  ),
);
