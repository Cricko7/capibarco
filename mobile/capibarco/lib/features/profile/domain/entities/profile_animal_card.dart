class ProfileAnimalCardEntity {
  const ProfileAnimalCardEntity({
    required this.id,
    required this.name,
    required this.speciesLabel,
    required this.breed,
    required this.city,
    required this.photoUrl,
    required this.statusLabel,
  });

  final String id;
  final String name;
  final String speciesLabel;
  final String breed;
  final String city;
  final String photoUrl;
  final String statusLabel;
}
