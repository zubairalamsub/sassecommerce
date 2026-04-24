import { useReviewStore } from '@/stores/reviews';
import { act } from '@testing-library/react';

const sampleReview = {
  productId: 'prod-1',
  userId: 'user-1',
  userName: 'Rahim Uddin',
  rating: 5,
  title: 'Excellent quality',
  comment: 'Very happy with this product!',
};

const sampleReview2 = {
  productId: 'prod-1',
  userId: 'user-2',
  userName: 'Fatima Akter',
  rating: 3,
  title: 'Average',
  comment: 'It was okay, nothing special.',
};

const otherProductReview = {
  productId: 'prod-2',
  userId: 'user-3',
  userName: 'Kamal Hossain',
  rating: 4,
  title: 'Good',
  comment: 'Would recommend.',
};

beforeEach(() => {
  act(() => {
    // Clear all reviews by replacing state
    useReviewStore.setState({ reviews: [] });
  });
});

describe('Review Store', () => {
  test('starts with empty reviews', () => {
    expect(useReviewStore.getState().reviews).toHaveLength(0);
  });

  test('adds a review with generated id and timestamp', () => {
    act(() => {
      useReviewStore.getState().addReview(sampleReview);
    });

    const reviews = useReviewStore.getState().reviews;
    expect(reviews).toHaveLength(1);
    expect(reviews[0].id).toMatch(/^rev-/);
    expect(reviews[0].createdAt).toBeTruthy();
    expect(reviews[0].userName).toBe('Rahim Uddin');
    expect(reviews[0].rating).toBe(5);
  });

  test('adds reviews in reverse chronological order (newest first)', () => {
    act(() => {
      useReviewStore.getState().addReview(sampleReview);
      useReviewStore.getState().addReview(sampleReview2);
    });

    const reviews = useReviewStore.getState().reviews;
    expect(reviews).toHaveLength(2);
    expect(reviews[0].userName).toBe('Fatima Akter'); // newest first
    expect(reviews[1].userName).toBe('Rahim Uddin');
  });

  test('getProductReviews filters by productId', () => {
    act(() => {
      useReviewStore.getState().addReview(sampleReview);
      useReviewStore.getState().addReview(sampleReview2);
      useReviewStore.getState().addReview(otherProductReview);
    });

    const prod1Reviews = useReviewStore.getState().getProductReviews('prod-1');
    expect(prod1Reviews).toHaveLength(2);

    const prod2Reviews = useReviewStore.getState().getProductReviews('prod-2');
    expect(prod2Reviews).toHaveLength(1);
    expect(prod2Reviews[0].userName).toBe('Kamal Hossain');
  });

  test('getProductReviews returns empty for unknown product', () => {
    act(() => {
      useReviewStore.getState().addReview(sampleReview);
    });

    expect(useReviewStore.getState().getProductReviews('unknown')).toHaveLength(0);
  });

  test('getAverageRating calculates correctly', () => {
    act(() => {
      useReviewStore.getState().addReview(sampleReview); // rating 5
      useReviewStore.getState().addReview(sampleReview2); // rating 3
    });

    const result = useReviewStore.getState().getAverageRating('prod-1');
    expect(result.average).toBe(4); // (5 + 3) / 2
    expect(result.count).toBe(2);
  });

  test('getAverageRating returns zero for no reviews', () => {
    const result = useReviewStore.getState().getAverageRating('no-reviews');
    expect(result.average).toBe(0);
    expect(result.count).toBe(0);
  });

  test('getAverageRating only considers correct product', () => {
    act(() => {
      useReviewStore.getState().addReview(sampleReview); // prod-1, rating 5
      useReviewStore.getState().addReview(otherProductReview); // prod-2, rating 4
    });

    const result1 = useReviewStore.getState().getAverageRating('prod-1');
    expect(result1.average).toBe(5);
    expect(result1.count).toBe(1);

    const result2 = useReviewStore.getState().getAverageRating('prod-2');
    expect(result2.average).toBe(4);
    expect(result2.count).toBe(1);
  });

  test('handles single review average', () => {
    act(() => {
      useReviewStore.getState().addReview({ ...sampleReview, rating: 3 });
    });

    const result = useReviewStore.getState().getAverageRating('prod-1');
    expect(result.average).toBe(3);
    expect(result.count).toBe(1);
  });
});
