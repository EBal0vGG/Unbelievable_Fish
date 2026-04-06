import { apiRequest, isRecoverableApiGap } from "@/shared/api/http-client";
import { mockMeta } from "@/shared/api/service-helpers";
import type {
  DealProjectionRecord,
  DealRecord,
  ServiceResult,
  UserSession,
} from "@/shared/types/domain";

export async function getProjectionByAuctionId(
  auctionId: string,
  session: UserSession | null,
): Promise<ServiceResult<DealProjectionRecord | null>> {
  try {
    const data = await apiRequest<DealProjectionRecord>("deals", `/deal-projections/${auctionId}`, {
      session,
    });
    return {
      data,
      meta: { source: "api" },
    };
  } catch (error) {
    if (!isRecoverableApiGap(error)) {
      throw error;
    }
    return {
      data: null,
      meta: mockMeta(
        "Deal projection is not available for this auction yet or the backend projection endpoint is not reachable.",
      ),
    };
  }
}

export async function getDealByAuctionId(
  auctionId: string,
  session: UserSession | null,
): Promise<ServiceResult<DealRecord | null>> {
  try {
    const data = await apiRequest<DealRecord>("deals", `/deals/by-auction/${auctionId}`, {
      session,
    });
    return {
      data,
      meta: { source: "api" },
    };
  } catch (error) {
    if (!isRecoverableApiGap(error)) {
      throw error;
    }
    return {
      data: null,
      meta: mockMeta("Deal by auction is not available yet. Showing auction-only context."),
    };
  }
}
