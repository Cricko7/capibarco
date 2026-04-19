import '../../domain/entities/donation_intent.dart';

class DonationIntentDto {
  const DonationIntentDto({
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

  factory DonationIntentDto.fromJson(Map<String, dynamic> json) {
    final donation =
        json['donation'] as Map<String, dynamic>? ?? const <String, dynamic>{};
    final amount =
        donation['amount'] as Map<String, dynamic>? ??
        const <String, dynamic>{};

    return DonationIntentDto(
      donationId: donation['donation_id'] as String? ?? '',
      targetId: donation['target_id'] as String? ?? '',
      targetType:
          donation['target_type'] as String? ??
          'DONATION_TARGET_TYPE_UNSPECIFIED',
      currencyCode: amount['currency_code'] as String? ?? 'RUB',
      units: _parseInt(amount['units']),
      nanos: _parseInt(amount['nanos']),
      status: donation['status'] as String? ?? 'PAYMENT_STATUS_PENDING',
      provider: donation['provider'] as String? ?? 'mock',
      paymentUrl: json['payment_url'] as String? ?? '',
      clientSecret: json['client_secret'] as String? ?? '',
    );
  }

  static int _parseInt(Object? value) {
    if (value is num) {
      return value.toInt();
    }
    if (value is String) {
      return int.tryParse(value) ?? double.tryParse(value)?.toInt() ?? 0;
    }
    return 0;
  }

  DonationIntentEntity toDomain() {
    return DonationIntentEntity(
      donationId: donationId,
      targetId: targetId,
      targetType: targetType,
      currencyCode: currencyCode,
      units: units,
      nanos: nanos,
      status: status,
      provider: provider,
      paymentUrl: paymentUrl,
      clientSecret: clientSecret,
    );
  }
}
