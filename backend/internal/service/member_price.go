package service

// ApplyMemberDiscount 返回折后总额和优惠金额，金额单位：分。
// 折扣非法或 10000 时不打折；最低折后 1 分。
func ApplyMemberDiscount(originalAmount, discountBP int) (totalAmount, memberDiscount int) {
	if originalAmount <= 0 {
		return 0, 0
	}
	if discountBP <= 0 || discountBP >= 10000 {
		return originalAmount, 0
	}
	total := int64(originalAmount) * int64(discountBP) / 10000
	if total < 1 {
		total = 1
	}
	totalAmount = int(total)
	memberDiscount = originalAmount - totalAmount
	return totalAmount, memberDiscount
}
