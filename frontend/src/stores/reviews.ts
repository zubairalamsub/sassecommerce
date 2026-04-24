'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { reviewApi, type ReviewResponse, type ReviewSummary } from '@/lib/api';

export interface Review {
  id: string;
  productId: string;
  userId: string;
  userName: string;
  rating: number;
  title: string;
  comment: string;
  helpfulCount?: number;
  createdAt: string;
}

interface ReviewState {
  reviews: Review[];
  summaries: Record<string, { average: number; count: number; distribution: Record<string, number> }>;
  loading: boolean;

  addReview: (
    review: Omit<Review, 'id' | 'createdAt'>,
    tenantId: string,
    token?: string,
  ) => Promise<void>;
  fetchProductReviews: (productId: string, tenantId: string) => Promise<void>;
  fetchSummary: (productId: string, tenantId: string) => Promise<void>;
  getProductReviews: (productId: string) => Review[];
  getAverageRating: (productId: string) => { average: number; count: number };
}

function toReview(r: ReviewResponse): Review {
  return {
    id: r.id,
    productId: r.product_id,
    userId: r.user_id,
    userName: r.user_name,
    rating: r.rating,
    title: r.title,
    comment: r.comment,
    helpfulCount: r.helpful_count,
    createdAt: r.created_at,
  };
}

export const useReviewStore = create<ReviewState>()(
  persist(
    (set, get) => ({
      reviews: [],
      summaries: {},
      loading: false,

      fetchProductReviews: async (productId, tenantId) => {
        set({ loading: true });
        try {
          const res = await reviewApi.listByProduct(productId, tenantId);
          const fetched = res.data.map(toReview);
          set((state) => {
            // Replace reviews for this product, keep others
            const other = state.reviews.filter((r) => r.productId !== productId);
            return { reviews: [...other, ...fetched], loading: false };
          });
        } catch {
          // Keep cached reviews on failure
          set({ loading: false });
        }
      },

      fetchSummary: async (productId, tenantId) => {
        try {
          const summary = await reviewApi.summary(productId, tenantId);
          set((state) => ({
            summaries: {
              ...state.summaries,
              [productId]: {
                average: summary.average_rating,
                count: summary.total_reviews,
                distribution: summary.distribution,
              },
            },
          }));
        } catch {
          // Keep cached summary
        }
      },

      addReview: async (data, tenantId, token) => {
        // Optimistic local add
        const localReview: Review = {
          ...data,
          id: `rev-${Date.now()}`,
          createdAt: new Date().toISOString(),
        };
        set((state) => ({ reviews: [localReview, ...state.reviews] }));

        try {
          const res = await reviewApi.create(
            {
              tenant_id: tenantId,
              product_id: data.productId,
              user_id: data.userId,
              rating: data.rating,
              title: data.title,
              comment: data.comment,
            },
            tenantId,
            token || '',
          );
          // Replace optimistic entry with server response
          const serverReview = toReview(res);
          set((state) => ({
            reviews: state.reviews.map((r) =>
              r.id === localReview.id ? serverReview : r,
            ),
          }));
        } catch {
          // Keep optimistic entry as offline fallback
        }
      },

      getProductReviews: (productId) =>
        get().reviews.filter((r) => r.productId === productId),

      getAverageRating: (productId) => {
        // Prefer server summary if available
        const summary = get().summaries[productId];
        if (summary) return { average: summary.average, count: summary.count };

        // Fall back to local calculation
        const productReviews = get().reviews.filter((r) => r.productId === productId);
        if (productReviews.length === 0) return { average: 0, count: 0 };
        const sum = productReviews.reduce((s, r) => s + r.rating, 0);
        return { average: sum / productReviews.length, count: productReviews.length };
      },
    }),
    { name: 'reviews-storage' },
  ),
);
