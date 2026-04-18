class DonationIntentEntity {
  const DonationIntentEntity({
    required this.donationId,
    required this.targetId,
    required this.targetType,
    required this.currencyCode,
    required this.units,
    required this.nanos,
    required this.status,
    required this.provider,
    required this.paymentUrl,
    required this.clientSecret,
  });

  final String donationId;
  final String targetId;
  final String targetType;
  final String currencyCode;
  final int units;
  final int nanos;
  final String status;
  final String provider;
  final String paymentUrl;
  final String clientSecret;

  String get amountLabel {
    if (nanos == 0) {
      return '$units $currencyCode';
    }

    final fraction = (nanos / 1000000000).toStringAsFixed(2).split('.').last;
    return '$units.$fraction $currencyCode';
  }
}
