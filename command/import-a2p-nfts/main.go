package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	bitmarksdk "github.com/bitmark-inc/bitmark-sdk-go"
	"github.com/bitmark-inc/bitmark-sdk-go/asset"
	"github.com/bitmark-inc/bitmark-sdk-go/bitmark"
	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
)

var a2pV1Assets = []string{
	"fae5123e72f981de8d9026c76aa31b5b0bc3d2c6609f77fee099931df71c2f246a8dd60707a7c05f67a267ab68f7c7d8b5e1f2d3770ece93b4e45c6122e8b371",
	"870a54c557095c8792bd0516a3836f5430db64f75498c2b6d0fed780aad7f1581f3d9ff19a506f10e4773c63f36fda3b8dec73b24a0f094d57098f31b7c828de",
	"1cc021b69c74d3d40f7968e696281f212e5e6c83635da7b2e2dffbe8765d8cdbfea51a222b19de0beb897177cf5bdf0021c141a67dccd7286932e1f5a3f73757",
	"0326cac8e599fd504e3494a32583079d6646a35b728608ede0433c9cf85f38b64f4467777952608f8f6070034a514cb796d1ba845fa866c2119ed5681d5dc771",
	"19ad2285efdcc462d287c95e52ca1957eeedf51ea1c104d7a69f5b5d43225b088692f04954a192e87cd97c959b9a95cb16915f114723c70a7e87afc5b9bc371a",
	"cb83ee6b1e032a7508cf6a7c24a04e2f0c3fee9aa845a05010c92208035b5abcd705511a9707d2ceef0637e34b751edf9393b206446955ec8aa8c034dd4b7fb2",
	"9231e1da3fb2378163d6081546e671cb8f2174e85dd3980c155b624f66b0a573a7d0b6c925c569ffd1cec1b955d33d9254d7ae1073dce86cc8d6f65e18353319",
	"16e1d9fac5b345d246328eb4e0a81d85417eb40e2047657340485239feaa463801282992087565032082869139b794514967c63d57e893505f1aab94b243e1c1",
	"c1d70dc786c5cc776bd62fedbc2d767d9000d834c237265aecc84f17e08d992dfc587f67b3ad3adea77b57270e9425d7aa71c58d5b9f44e2f8850b32abf05846",
	"22c27be4963aed865e9cccfd2c8688459ce479cce7f4b026c1ec3104634e311770f1f08df343e6cc08075281ab3b814c61d990ada21e9427b2a7864968cb6a28",
	"8f9cdba5e8702cf4de1f27ea7aa6448c839cffaa11653e63d7c074d0ea502190269bd8d291cfe422c1f0cd4db2088b4d641b0883f7ea1b7f276a910212559e9c",
	"00df78550ce3fe8c6702315dbadc4d44730ff16b4b30d4da59bdc1cfbc05f50c5b64be49be68534dbf307ef8278e2e446987294ae5f8491c72e791321d49e436",
	"e0ebfa6459ef1bdf2d8b280c17eb38d412b308a8df92042a4ff093f4c0e874615edc7878c24d306a56c0e46cc6b0f0c20a8ca2dbd88a30416cb09ef419723545",
	"21960fa9daf2c903101b3f39fa78377ab60aec3b8f67282da8e5380f8523e85f3cad24cff2bb4d51ed367d33a6321e286ca373964887511cdd874aa3d0c2a932",
	"fadff45c2d2c0fd4258b09276cf484fc84e78daf454ae9f67393809471af9ea4900ecf6918e37290fa180ce4346cdeb9c47c53ab5e3dc8959008a062ded6daf2",
	"9790971ffd92886bc1b8f263e6db0e81605f76d1230d887555baba9bb39cf7f8dd1fbf7c6b737064720e277fe982d3a319d204d3b6cbd155d74ef12ba594f462",
	"4e6f58cd2c6a89374f0e41402701978ba0e40a5a34b525490cab02a57129b313f34f8f492c377049a0dc52908fd9b05fec106184a962bc1f5ae747a6e42a00c3",
	"00512701d7ba461b61b965e02568ce12634a3fe6cb964e3de2d13d8e895a126f5a3ec10bf4baf44bc733c1958c22383aba3047f1b9fb8f8786c11f3a422d85c0",
	"13a4b308c30e937df72f865fce46d0a867f3568b795d5de4eb9bbff85dcf0a95d1596bff4614df0d1f68b86bfaefad54835146715619dd8a91c82a7b0d742e87",
	"0c5b384874714b782906e918bc9f56478126831cad835f349edc67fb4565974b37603ebbc1da97980afdee0ea8edf5061b49512523a1e710666d7fb10f9b578f",
	"ad8c6ce1a6ac30c595f281f3f776996482d9c89b08c14a9da82384e92ce57083277560de4f9c44c372ff77bedcc5660394a8116f240b09f06376d3074ff98447",
	"fdefb99e7e97f774750e808b5fac6487013979d31b01c43693027ef8345f716b78fe8934ceaed102f1d2002469fcf39a208e29cf3d56060806113b937832e4ef",
	"2be2b08fd71747d524ca39cdb72cc1bbfd6692759eda6cb0fae8f97f2d3b117fec03539753bd2ee825f0a89bb88ffaef2d1dc0e77acee9ed64db0a14970b4f1d",
	"0ba03b606b6980dfd7e580d7158d8b8027c55e8768af62921da43987043c8646541ea26b9d6d966fe9882b78ede5f6f97e738459afc6c8af6796b9eb6fb39e57",
	"d27dd15b9da4fd5ee26e7cc0fb617c12c787cc51622615e3772f0f17a2505a7bd94fbb3a4e6d2b70529c46b1e944313c09fb735b8ee11a55ce8560072d091d06",
	"584f2f47d9269ef0c9fb162c081ca34f6b964f8c1a4410b87633a5dafa5ff1f1d4558021ab04fa2566e066e5ae474dae3fbd7dcb9a0af799491550c2a3e88968",
	"c0c24f631d658361b26cc4bf69a377cb483a5316284a29df08534698763d3838b67fe61878bf13b0bb33c9fdc3b07cd657dfa5a35c79b6a6cffcbf08f8cb2cc3",
	"cdde8d5d1b77892938f1d3aa03e02a220870df64dbd399e054b90ce0550fd7c348a4283629c1873d6d05ee401b2a80b121d44001fc8438885cf7f2f202386093",
	"4d7a113b35e1435ad8888a8e590a9c6a7aa5f5ab04fa6d200d17580991f5da37644b8c4cfb7f95b5da7e04a4493215789d1325efa8713b2cabd483c08d16a882",
	"2b02bc0d9546ada0bf5579d6206ae934b3e87df2bace2e858c882b2683bb2d678b5f3269c5a189bc63ae174982eda462c278918129cc877aae3c921e25e504fc",
	"ed4eafd73a126e1fe7a9b7fe0f43bbfe895b994ad42267b7bd25764b51a7df78f2432b97a9fe37f5c6ee6e702e473d9d7dce22ee5042d63ca6b083f471f94b25",
	"bf93a2619a3cda016f3d3a64714ec0e6f65c5a2d23be8624d91e9ae12984aa53e708851e25414b9fa576d016297f53435dbfeb652ba0e79597ec63015f5be182",
	"9532755a7632298c93cdd85498770c7871fc47e7682037d0c20fe03c4ddfd4cdac2dc46bc1ea00acccf9d7d45596cb10ebe452e532f6c5b3f7f49b78806d43a3",
	"b7e94a8ba42788102059e84116f91cd7d95e2a1d4536616650376dc05693dae60fe8a20975cfa5711c6e1baac27d51f149f4973f86aa3757632b2701b9429d18",
	"a3cab2b1e1625d70ddfe00cd541b6f5c18e888f13573b757804de540fcfe33c91a6040c3644ed0c57dc1e597bf4b461f417077aa4e5fef2a40283ed44bcb0a73",
	"2fab8d31619e47635281238e10e81fb7d9a3657a84437173af9465107073d99634862d0b7b438d10068d286340aa3a06bba0e3db0f9b1e86a4ef59f2cca69423",
	"76dd1919a3e8244a21b2922d3a9bbb3e98780c63d286e7585073710aa9db4f110841a0f4a36dcd0f578a14230de0d171da18324cd832d003550a2290ed6c0ad4",
	"9e04bfd1cddea4e8d60350ac9377583e177806876fac1e2c35edb3b40cc4a0cc8008c309c6368fbbd840488f3fd3b373d8b4f3f0b9883385ffb7e6e3464fcb1c",
	"24d5b6af0a2ba601cc4caa332ccfdfa5a3b72feed8a934a4e5247874f6f98f45361d9eb0980f781f212fb24cc1b422210fb5b82992a0a77bf6b5eba229c4c9b5",
	"77d6438e13860e074ed50db7ec19cf7f12070ceff68d834db17bc11dd729d6becd778d7922576f5eda971c49bca944693df3efcb5ae0712eaf420742cc97e8f0",
	"a8515df57262568c3107c19e6453d49fa0e8175c5885ec85949a175dca48cbee968e6a1fad86e7b2de42018e6542db27426b53cdfac9eda7c894e998d63ea0df",
	"af4d517f77cc70ed54b56cd9950ba1488d3fdcf6325f95c4d0167713be09d432fd4d550864c310d7f4d6e44bc90b4770d008deaa3986c5dbffc2c1962ab31aa8",
	"6b1d9ff1147f27de0b1012654c90272231566aba3f3b4df68fe87bcd54dd21178c279c667ada8b386325f39538fde5cad08f1cbf7b7088d8f4fa044a42482ea0",
	"0cb38413ed3c20d254ce191d116b567e19a7d75d0959330397f3a8c175831f8c655adcf5b2d275a47f4c25237fa53d46930222c611cfff7980042a2099e3d8fa",
	"b0a53c95f08cd4beb48f9bce8ca46c82eca5e669f644c230eee253ea9900e6cc5ecd5b05e600b66f950dd854f4b6acca2c4430b64a34681379fc709c5bad79d4",
	"7b37253fcbae3407c540c3e07bf850097807695ee3bd21072a42b1093bbe76c4ed6b5f54aa4a415521cb401f21093781eb426f9ffc6fa7d68f090d0ee80c3a6e",
	"40e62509321e2eafafbe6f017563ed799c1c52933d9ff5f962ebbffac7a59d29d756b5499b3c59a051f00429bd021d8f5988925e984c6b998bd7faa67f5aeb4f",
	"e4a28af106879098dc2c0895b552a175354d8efb32d54b9564124d8da73f1230255810fb504c8cbdb76358a902d74198420787b43f73ab8aa22896eaa96dbecf",
	"24e0bfa0607cf11478bb87a1525e4e2b80121688917892bd837cc463d4e99ff756927bd6d5139b1691334e386870b47392abe604a0aaedc68070a212aca24556",
	"85c4a7546ce001f499be7f32dc10b9911623f225cae9ad6875c0de92c38725606a731e57a4da8586d1dcbe95afb4428d090352ab6706a39a7208adb0310d1a69",
	"25f703bc24b767f92c8ae986cbfaa2b69cede3cb35081708254125adb597d3f3bcbb7bc0efdab17ea4d1d8b6690516fb48c163263ea6e184bd631969dd9977dc",
	"e073d5803b84b9cfbf8469d59343cdac3366888c481aa9e695200da47f1f471d7c2683c13ffde1843db30e0e405ad907423081bd7d4be0f44ea8189912ed56e3",
	"575c1b9369b780f461ad59780f758d11001d83bfb36dfb153ef47fd411c78cfb4aa1ec1b88681c520b547812b4c28c158e8f4351b1e9811d01d5b794cd11cb40",
	"d3249ecaaa857d8774db1ad51b60d64e9be6c088a24053b5ae4df0c452e257f92b77eafdcd36376793ed85f413b46b7f771dc6a1ee41325b413163eb44ccad88",
	"0d152a968bccc0031b19023e284c0b544d698a802297cea06d4dfac8835a8a92bcf3f457e2fb4f0c8927e177540bcc9329cecbaadf21f896c981b667eb265b68",
	"fdd43aaaa5c7a42f5d5d05dcf29c1f2425dc2440cfe9f636dff5994311939e73238d85487c8b79f6ddb3ec66c158808b44698d228b7679ba06e58dfde563c440",
	"8b091f09bbe1ba4fc4e5a49bb1f90e7eeb373fd333af8118db9e35c6447c13a97e8041a79ea3c0f837cafc80ecddf2b7b3be2fed1814cca9978f712639faea64",
	"efaabac6796b513662c9715cb59465f485da20c611590adc0f0a6f7bc4dda6635555cd21c88432988b783b15e97084a97b9f109fe21b07491a6416666d7f9211",
	"34d86d16639fbe76b1c60b6ec4a2d86aa5d354ed051a24fa6c1eb73906ea5c8a76478cf1159b5959fd88882e9d36d6da600ab81e2edecbdccac192ec674700af",
	"c0a8e7cc930bd183ce1f0874ff627b795fba7752b35848dc1e39be7c6953c12478f06b9128f3a7de4bd8005719732a105fa86374374e7f473e6891b9f2d5aa9d",
	"ee012f2a98b5a78b6beba86f1e0f57309b31d24ccb433d9a53ae12977bd9aba580ecd3444c2245f3e808333ac7aa5c1669e0630bef80e7b2dad23673a2a0a5e1",
	"7a114d1f6d58da50458a63e9eafddffb1dbd9bb4e52c24f96c01b7309b391f608691764743cf677990036ab4671ed7eba907edefe857ca70831f1679b08a8ebb",
	"9785d46093bdca031543c1233bd5365d4b3f160128a43ec7230522e3eaa0ddd50fe008d99900d1d08153501ce0d6d29eec65c12d3345aa5fb640bc481aeda44e",
	"12b4f65a7bd44303cd4c609619636754e989d06b8cfaae74abc3088cde59c3df85deff10af7d8ed3235093bfd0143d4fa6fe8a663e76583205bf387a4439ef29",
	"70bb83dd3d6d5af0e0fd6fe4089a6c0654159e23d0a6c812485a04b671924d9d13b08b4682119c3fde22ed2501f7e7a51a0bdd90d7f7874aefb8751301f67345",
	"67b3339f3aea3c2aea79512e0b7222895bd18440bdbcc4dd338061f30e6dc09413f9ddb0b603bb668ad6212d1c53a65164e399c36c5b1640694375e37518256e",
	"cfe191382fead271371ba59cf9947d4f0a11f6c01624c623027186ac3db86f88e4fd792e9e0fa224a1ba17b8c23982871da6d8a873ef625c92eb63e60dabd61a",
	"a0e1f6abce15975b20b5c904145ff165b9e2994a0bffc85af4e2a8b1ba3f714dd04847e9f2423b6aa2209b990e6880165e8f37566577629f866e586b114144ba",
	"25afff61276e637ca7266669eb6e74d471f7905ef39f1cf668b769630c59caddaf38cb7455c963d798e6b3b3699ab268247713b0116a429d75fac3a83e3c5c28",
	"d5d1160b2989857566a7958c96fb6d318387078a3afd47b275c838aaf8e0ce9f204911329acd5f8a7e606595a5fe66a86361e74ca43f6c7b7a2097d949fd3299",
	"51a7223d5daf8123cdaff1950bd524ca3e8e9f9ff9ef8212dea802787c35143a102b543979c7b449aaa587ec9ad5ad0d79df2113f43edd1525ede470104938a9",
	"38e0bdc40b421e03d5110c38bfa72a2bdfdd0f20e52ce62f313b9949564a9deb537d60df53612d0de0ede8243025a5b5ba9831af4eb4488934ae1497c28238df",
	"e186ad7ce73fabe0497d0b11f25d15be021494b6111680075b04cd3eb1f3e74813cc092ef43cad2e5395b426a2e045b1a93f01896a47b72a3fb4a5ab791d9c58",
	"80ccaf4f766aa4f1a0129dc65ab6c0dda2fa2101e3d1bdddc42b0373500fe3fb4348895f4d0fe791bc1cad066d8b1364b8d6aea4d686602eb5d405e441a966c5",
	"77141fc596448aa0aef524c8fa1744b06bfd81d863f440a57796cdcf2a7a407bcb4f6c797bdfe8331436e2b482664f9ab1341789f72a516ebb49342cb9e9d08b",
	"f2766c2c943999389a3f7a541f527d6fe372d279441d5f047c29f29d58a6b14cf48af66901268430c7bbac998c2be89dea9664dd3365fab2aba49e7d261a8100",
}

var a2pV2Assets = []string{
	"1df8189d61b6efa18ac25cd8f53a4f7e7aee4604ba9bef9092887f6e5d18e527579302421a3a05790684c3538fb3cedffb370f04daa98d5819d968e84c3998c3",
	"5854b9f45613aa9ef621ee7cdb9bc91caf0f8d096af5c7a26ce21604da7f43c0c7274d33ad6be4231707417d62da42adf6af9371e912c7d6427139bfc6c4bb72",
	"ec1855c65df0665a68d62adc469f2070d4292854d4dbaaaed5245810d29515b9ea86c75887a05296832d47abe5d4ca09dd6dec2a331ca035faac40d7f192d77c",
	"9b7e8260b06c8bf3861523ebd8d86b8dbb76ec72bf4e7211ea095f9551199b03e7e35668417260e0ed3ac6cfdd50d9a6c310a20f5c2592f1339772695dd6b89a",
	"f56c61b956caba01985c55ac5a6529e957bc2e82a450cb2c486591e3f83023ed02d2b4e74476df6620a86c93c7dfbd988161de918923690f407a047fcf3b8e84",
	"bbbe388357c36daf564eff89f27882f8af7a4f81b6af4584a40a9f02aaa459614fa2bb60084fafef235544ccb3800f9e0c340a6b0d0fd27eb6fdbb042b8a12f8",
	"5569f0aa2a8326b70840876d4cc0c27650b8c2800001df8251349bcf2b18d37adba26573a8e77642a675b03c4064bfccac964257dfe6e4a9278fe028c6f0ebcc",
	"38bf7ba0e90fc5cc5b657d4e951fc1a46a4bc824cf3f9d4360e3cdaac5803eca5703555aff9a113dec9cdddeba3cf167ec025e35ff37cad783ac9329bca4f294",
	"caac6380bacb5cc074e50dc5540a4e8d62106eaf1aaac24da7894a26b5d080662baed0474a39dcdd2ccc8395ec7442fe747988a0c37e7db785bf085263b21be3",
	"d95b7cb6bb64a6f76e54045eaf401a066da4cfdb59f2aab1d186f8dc868f88da5e58236ed244886977343d3c46b3ef739a68eee95c6982d215228698f94ebc83",
	"54e95076e0b85c40ff0e48305b6410aeb478c5ecfa9b6e3c90a9170600bbf5985f44f1ceaf32c896d4f013308495cf374c9cdce145ed9cb7d195d6ecbf8d589e",
	"a21cf5ea4606dc10d1ebef9868b58c80a66e8b36c6d99de231530000c575b1752587cd7916c5c109946c62507cf5e05aa74e6373d091b3180aea6d64fbc084ad",
	"050f05ec4e1bfb0ce592034809af3ca05c1912a1a5e6acfc3726e583a7f6024195d9e16bdc6c1d3eb1b2db1ed3d49b20fcb413f75d445979a18601ecabbeb8ad",
	"2662808d06a5de6763b88f334e565da92962c1e51dd69ea278c1088c9730a3e9039aaec20776e8bf9fc59034fdd8e40563bcbe0f4cd34c915d913d844b850bb0",
	"732b52e795eae5d44df9c52135e05a6c4cbe42213bcaad28440e9511b1484ac7bfc2c9f308b19dce84cda1d87fc17a4641b63c799e34e1444e6e06fbf82c56d7",
	"a0f3a31eb81b318650e66ac4f7a9f91223d41fea6d51820681e90183b90c3ec4ab941eb64ae0a72f307bef4bcdef3ca30af162af664c9d1e89db17fa30a24229",
	"fc8afe74d62333e186367fbdbc83057f827db3a69c6c1198d16d9c72d5df0e20506f0d86ff9e7fab77a3e8c03cd1b4e86dcc957a2bf672e39f7d77905b8695e5",
	"bf3517316b0ed6274ec39ff102c86b4d8818d2bff55468c355e6f816144b09932e85c4bab53d37b650f6eb835cc5a52420e0d038f2fc211410cb20333c078d1b",
	"f29ee5682c3f16a9357e3acf89dccb06cdd12dd6e889e6b8000249c0757449a18ff998a3fb84ccf78b30bb9f6c87d0cddafceded5fa159dc5c972d0c82260b11",
	"6776b862ca68f1721960dd266cdc1abdf432bd2bbc718aa7e59548e9f54d3901168310bc549f2f6925e107d367762c3ba318e996e24bd784c7630ea56620f652",
	"ecb84d0cd8f929eae038263ea7046beb54b8241f6e13bcaf2346f1ad6a92270d90e1006f1578d9391fd422c6d30b6067aef5cbef1d37aa182ed0798b3dc771f7",
	"2a05f6608798fdc1ee0bebb3beaf43c0568bf0f2bc0b12182cfae45abd3a07d1e650c059d49ee189b23af2f309b9a112ec27bf3fa25c23e51d9f42f9246c9091",
	"b9702798114b94c64c25cfe139d89203f05f20c890d5c1ed0f990d1ef4d0407bc896de7ee8acfc07de31bc2297ac43bbb697b16d2fb2093a5c6471221d3736ea",
	"c37b88725758a2d30cfc20eaee9d272bebe500ae600f36882baa73afee19cee654011cb67477cfb7741f02f428e9516e2e9b26758b465fa9a3252aef7f5dc6f8",
	"550ddcd62b56d84ab3d01b7d5fe8370efbbd18f0f446ba1393ca3b7289e0eb3eac02a097b68d989d0dada690497d3680a4e30026b82d230ae3dd2d9e58f991fc",
	"bdd5525ac9f6cf91c08b005bec44cd466d67690a9a62f4a6d2a1107de1008f67b053cf99c4053c0ac468278efe14a9840f0d959b63a97dd99430e67cb55cd679",
	"1cd52c081056cbd844f48865f46badfd88d0f4bd7fd919b77b98d30978332b5a4aff3a962004636a78066a2b5d475046da97626b7b5bcbf45ac1c16fca5c76ad",
	"288d9b8f1c29c25aa14e2d681ecbd1712af2e89d4f8aa947708dfe1f9a96447a812d7ad4f4eb50c677dc2e9c11552baf3c137b22ce4d9e4ee8842e8308994071",
	"f7a8899212a3c89918743fb38a4f70abb86869d97a2e7dce90633d13be73076e176b8a09f4531bbdd167214fef77fdf89f6ec3afb0ada200f056c1b4e0a8d58d",
	"b53e975ff6ee71a5da2b7590408938916005e97c85a80f8197a20ad7a3adb73db70d7a100276ccd1dd3b5c9a606a7139713b92231bdc684afbdc077f45e1006d",
	"e176787c46fd96b11c5dc24a31ad5f43cd1176bce38d92333afdb18a98782f150970105e86725eca8509d99e174696d83fe77938d92a97a3f43a040e4ffe5870",
	"70db7e15427a84a4ee302fc38a3394cfc6d86db576ea3e4cd1361b5c9a6d273c963d84412f1c184521147c6557396e3055df68576fe168bbcb94abb5d497b865",
	"234c929ff4da6a3ce30ac34d4a0f0204a57990c660674bfa5737990f36a1f54daf963de260685dc677704b3c0fe5a2a78d1bec81ff553a86badde6c4ce94e23c",
	"10615847daf05440d85a4136c24acc8708c3ceb1f0e9dcb7becc42329b0ec5ac72545dd0c59db34f3e271896bd827afd7d0878f8ba5c7b48cd4c20fae0282d79",
	"9caf687a83b74989ea0bfd2ef7d5164471454aad757405afdc3707308207fed86dad5cf6cb60334d9eede27af972dcd25214153494a0a7c4aa05617d4c108952",
	"ebf09a25c7d57d1d27cb206018071c20b34870ce35f36d092b65412eb987df29af4ea5ac3ab91f092ce849bd1fc66133891bedf156d102b756f2e115985db1a4",
	"f25d0825f2c1a0d496196339da8097d32657acc4622d0a53412c0ac5b1abd4565a464103a71b19757f8c24304c3f94d9f2c76755ca813c3cc231398757b51ce6",
	"6d11fc5e88f24e3e0b60468f19f21d6052d02131852636c0d4f02334603dc37352d96b84f086a54e483af28f6cab3f286ba2c25dbb2cb6cd50d0f3f16ca6900b",
	"133adee12dd23e0ce393c0eb066ba42f6dd4d053db1922eede7449b3bbbd0662310f65e602b7ba258eeb0946728aa142d9da86c04e9e91d450d878837750a6f8",
	"959c96d6085d60448a8d20fb55b46c3edf72eeda847b421d273406f93b53c92dbadf79d7dfe214d4afc1e46f9f6764c8197499cd99c4cdc4fa8b2fbcdb21c278",
	"31c3c23ac904664a54c3d454256ec473f760c2843dd3ea145759f6544e2e78d1594b1df5b2be2adaf761b0a48c04c2ff7eec7e7fbec6a8403a944de7bc54d061",
	"27f19c6ca23505fa501f12ae6c4a5d96446f265b6cb87bce7a5d9df4b55cde4ababca736e2d720200869c777df127de842641a22a1deb944ddfbeddc01f8f249",
	"de6813676aaf72550ad85cfce5b38ab903edac2d85a66733677f7d3c02970ce7786ce35ee692deecba44679d8dbf30003f77697fe5060531c2fe8e81007d4b91",
	"d49e3067884dd234ae6046a3cd3279a4f9161e525df99aec51479b62dcdd4c4cea0bdc8979a9284e9b82ca4a6a662a16f9493b734cd8815e2932528d9c181a26",
	"4688fe4dee60228fe429271c622dd48cef51cd888489f428fbe91ce170705b1f0f7028707cd0f028fb6e8a9016c2ae6a3551851cdd5e0379db565802553ce984",
	"d86510044ff0a29859d5b7c120cade9fefb84663deac2223c8b93cf4916370c7b251805a84c083823df8ba8413074324e041f8736c2c95bb97784bbc73186708",
	"326c6e057e5bd48c5edae92e7143085170d61a9de0d1699b2884c0adda89b49dd623542aa60575eba3a4b27e23b4cc96403d1a39743f835b28b7d7410691ef89",
	"9e9b1f6771a298a4de8a26e549c51925c4783ebad621ab7e02ec227afcedd1ebedd58e2d2f9e268dc1734885d2bb0ac8605bc8f2ce8e6ca40ec6dcd6d9b0b61f",
	"4cce92b17cae6b52114ecded8d009ccbcf1984e68d315e2e5870a13d62bf766efb7b4e4c09e6d7a214b650cae40dd6ef0c6ba12c4a56ba6c3689ac4eb45c264f",
	"1ad3cbbc0cae27714facacaa2d751de0cd0304d4ab62baeaa2df4217641b993df7950c70ea05f47806e9a212eb3b3fb4c4e360607d77f4a05edda0a62dea7a57",
	"616b26070186704395d07edde1a1a37d6edd42e3852a631578002a3de98d0101c928cea446950a6086db92ebcf0854b47048e1a0300b558e96ff32b6fc7a0266",
	"62b09ea6e087dce77a9266c9b8e70528e00c9694fdeb4f44f1ab880c04d3f8603021145472eab78482fe5aeeeeaf0a732187b37e327eac53adbff7f14690bb12",
	"0f5365f92a7b9bf83d4c5654ed1d003d365d2b841e72ffa940371dccff908d0be6d145d2a98d3d007e111c9464fa82c2f692698de41c23869daa4cd37582a131",
	"98a5e879638cf3b8a5c58e631fa0110d1e49ba74760f07dc6ea90f2ff67bb3b3e573a6f829dfc8af5d6aeda220095e8edfc43b132099221598b74f60aec18c40",
	"78d644145f9d36aa6c04fb2fc40c0c6cb8f52986e87bc6df36f2156a69283ae30898be30dc91d12b2d4c6baa51526c755b1f3a3a70dcfad8ff9ae258b27b5059",
	"27160d0e9af9c43d2f5ec15c5d3c7c4954bb793074ec73b18ed13fdbefcabb9c04278fc6078560c460831ae87f79291ab64d1074b81f9dbb3ddc06e1bfbc2772",
	"930be3734f87c7691dfeba556304bc41832afb607389d2a6b0bb1286a43369c3f3d7bc1f2068e89227956c5227df67eee96e359f02ad13a51027d9d2285c7ce9",
	"ec08b5ce0e5a0921b7a4c6420501534711f907e3868756f6df64e315b74806c552102e92b8e81b512cdefc0dcb5c089d0463a9098805271dc3801709879ebb3f",
	"4f7da6e9c69bbb0a019d456a9cf31bdad491415610d9217d696e74a85444af62a84f7e741c33488301534631ed2033aac3a1bbb25e0e5bc07c3f2f3d0a35a9a1",
	"b6190bf55125151f660d285afec922c15b4452ab9b2bad2a59adbf679c43ab4da629f3b4d471c93e80fc2578253c8d97e51e93ca23443eb8b25dd7600b259700",
	"7ef74436c9b216944be1de2ff97357b8c7c7b6319ab217616d5b91f13912acb791baff41a55d831a4c49c36074d67c45ce779288a0b5b6be18fc0f2fbe0d3615",
	"e6125ef1b2c79675014f031cbb8e6e9d4a8568f397c734f7cc046c5c86b53931c35fe780ac170140f1067e972ecb4df90108bcdedf370cc928bbf6c4b60498c3",
	"47b6b05af1a3329a87ad345c715ab1a6479f6d5cbefb161642bcf46001ba9fad18750c683cf07c6247cdbae6c5e087021eece30a6024bf69d182995adde166e7",
	"826cca47aea469d572ff7025eb4082bbe30686a793a2d689605a2a3fc71ce49cfaca2f6b6db35c3a6318777bbb94aeaa5559e9348912b5593821f19eea712a16",
	"a82f6826072b677b015333af737b8f4d7828b010113881ec1d54c085085cc875aaaa0fddb026cad7fbe7f45045c783fb10225ce06d6cad99e708ac7b38147123",
	"be540e9c5f7dd86e447ee3927bd1e8db891cb32cb173887246e71aa82c634b22a7cde4d1963fee91d77b96ca4a44ea06ad2f211dabcd8cf466d290f3bc2f1ef9",
	"2efc5e53fcf5511884248c9955b1cd2aa9bdc696fc313295b2357316c7c4366fdd495bc58dee5f119e50606fc2683554e4bbf5a888585d86759fdbedc74ed2c7",
	"de29faf9b7d4a301e5d6b208116584a3c5d5af47dc5fcd3edac05a6440e0b3f7746108b6a7c7887d96f4d83692b22637fb08a815cc45f83cf9b5895dd2360a59",
}

type Provenance struct {
	TxId      string    `json:"tx_id"`
	Owner     string    `json:"owner"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Bitmark struct {
	Id         string       `json:"id"`
	HeadId     string       `json:"head_id"`
	Owner      string       `json:"owner"`
	AssetId    string       `json:"asset_id"`
	Issuer     string       `json:"issuer"`
	Head       string       `json:"head"`
	Status     string       `json:"status"`
	Provenance []Provenance `json:"provenance"`
}

func loadProvenanceFromV1API(bitmarkID string) ([]indexer.Provenance, error) {
	provenances := []indexer.Provenance{}

	var data struct {
		Bitmark Bitmark `json:"bitmark"`
	}

	resp, err := http.Get(fmt.Sprintf("%s/v1/bitmarks/%s?provenance=true", "https://api.bitmark.com", bitmarkID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	for i, p := range data.Bitmark.Provenance {
		txType := "transfer"

		if i == len(data.Bitmark.Provenance)-1 {
			txType = "issue"
		} else if p.Owner == "a3ezwdYVEVrHwszQrYzDTCAZwUD3yKtNsCq9YhEu97bPaGAKy1" {
			txType = "burn"
		}

		provenances = append(provenances, indexer.Provenance{
			Type:       txType,
			Owner:      p.Owner,
			Blockchain: indexer.BitmarkBlockchain,
			Timestamp:  p.CreatedAt,
			TxID:       p.TxId,
		})
	}

	return provenances, nil
}

type AccountAlias struct {
	AccountNumber string `json:"account_number"`
	Alias         string `json:"alias"`
}

func importTraderAliases(ctx context.Context, db indexer.IndexerStore) error {
	var result struct {
		Traders []AccountAlias `json:"traders"`
	}

	resp1, err := http.Get("https://a2p.bitmark.com/v1/s/api/traders")
	if err != nil {
		panic(err)
	}
	defer resp1.Body.Close()
	if err := json.NewDecoder(resp1.Body).Decode(&result); err != nil {
		return err
	}

	for _, t := range result.Traders {
		if err := db.IndexIdentity(ctx, indexer.AccountIdentity{
			AccountNumber: t.AccountNumber,
			Blockchain:    indexer.BitmarkBlockchain,
			Name:          t.Alias,
		}); err != nil {
			return err
		}
	}

	resp2, err := http.Get("https://a2p.bitmark.com/v2/s/api/traders")
	if err != nil {
		panic(err)
	}
	defer resp2.Body.Close()
	if err := json.NewDecoder(resp2.Body).Decode(&result); err != nil {
		return err
	}

	for _, t := range result.Traders {
		if err := db.IndexIdentity(ctx, indexer.AccountIdentity{
			AccountNumber: t.AccountNumber,
			Blockchain:    indexer.BitmarkBlockchain,
			Name:          t.Alias,
		}); err != nil {
			return err
		}
	}

	return nil
}

// importV1 imports all a2p V1 NFTs to indexer
func importV1(ctx context.Context, db indexer.IndexerStore, assetIDs []string) error {
	for _, assetID := range assetIDs {
		asset, err := asset.Get(assetID)
		if err != nil {
			return err
		}

		bitmarks, _, err := bitmark.List(bitmark.NewQueryParamsBuilder().ReferencedAsset(assetID))
		if err != nil {
			return err
		}

		var tokens = []indexer.Token{}
		for _, bmk := range bitmarks {

			provenances, err := loadProvenanceFromV1API(bmk.ID)
			if err != nil {
				return err
			}

			tokens = append(tokens, indexer.Token{
				BaseTokenInfo: indexer.BaseTokenInfo{
					ID:              bmk.ID,
					Blockchain:      indexer.BitmarkBlockchain,
					ContractType:    "",
					ContractAddress: "",
				},

				IndexID:          fmt.Sprintf("%s-%s-%s", indexer.BlockchainAlias[indexer.BitmarkBlockchain], "", bmk.ID),
				Edition:          int64(bmk.Edition),
				Owner:            bmk.Owner,
				MintAt:           provenances[len(provenances)-1].Timestamp,
				Provenances:      provenances,
				LastActivityTime: provenances[0].Timestamp,
			})

		}

		medium := indexer.Medium(strings.ToLower(asset.Metadata["medium"]))

		assetUpdates := indexer.AssetUpdates{
			ID:     assetID,
			Source: "Bitmark",
			ProjectMetadata: indexer.ProjectMetadata{
				ArtistName:          asset.Registrant,
				ArtistURL:           "https://a2p.bitmark.com/v1/artists",
				AssetID:             asset.ID,
				Title:               asset.Name,
				Description:         asset.Metadata["description"],
				MaxEdition:          int64(len(tokens)) - 1, // deduct the AP
				Medium:              medium,
				Source:              "a2p",
				SourceURL:           "https://a2p.bitmark.com/v1/artworks",
				PreviewURL:          fmt.Sprintf("https://art-trading-assets-preview-livenet.s3.ap-northeast-1.amazonaws.com/%s", asset.ID),
				ThumbnailURL:        fmt.Sprintf("https://art-trading-assets-preview-livenet.s3.ap-northeast-1.amazonaws.com/%s_thumbnail", asset.ID),
				GalleryThumbnailURL: fmt.Sprintf("https://art-trading-assets-preview-livenet.s3.ap-northeast-1.amazonaws.com/%s_thumbnail", asset.ID),
				AssetURL:            fmt.Sprintf("https://a2p.bitmark.com/v1/artworks/%s", asset.ID),
			},
			BlockchainMetadata: asset.Metadata,
			Tokens:             tokens,
		}

		if err := db.IndexAsset(ctx, assetID, assetUpdates); err != nil {
			return err
		}
	}
	return nil
}

// importV2 imports all a2p V2 NFTs to indexer
func importV2(ctx context.Context, db indexer.IndexerStore, assetIDs []string) error {
	for _, assetID := range assetIDs {
		asset, err := asset.Get(assetID)
		if err != nil {
			return err
		}

		bitmarks, _, err := bitmark.List(bitmark.NewQueryParamsBuilder().ReferencedAsset(assetID))
		if err != nil {
			return err
		}

		var tokens = []indexer.Token{}
		for _, bmk := range bitmarks {

			provenances, err := loadProvenanceFromV1API(bmk.ID)
			if err != nil {
				return err
			}

			tokens = append(tokens, indexer.Token{
				BaseTokenInfo: indexer.BaseTokenInfo{
					ID:              bmk.ID,
					Blockchain:      indexer.BitmarkBlockchain,
					ContractType:    "",
					ContractAddress: "",
				},

				IndexID:          fmt.Sprintf("%s-%s-%s", indexer.BlockchainAlias[indexer.BitmarkBlockchain], "", bmk.ID),
				Edition:          int64(bmk.Edition),
				Owner:            bmk.Owner,
				MintAt:           provenances[len(provenances)-1].Timestamp,
				Provenances:      provenances,
				LastActivityTime: provenances[0].Timestamp,
			})

		}

		medium := indexer.Medium(strings.ToLower(asset.Metadata["medium"]))

		assetUpdates := indexer.AssetUpdates{
			ID:     assetID,
			Source: "Bitmark",
			ProjectMetadata: indexer.ProjectMetadata{
				ArtistName:          asset.Registrant,
				ArtistURL:           "https://a2p.bitmark.com/v2/artists",
				AssetID:             asset.ID,
				Title:               asset.Name,
				Description:         asset.Metadata["description"],
				MaxEdition:          int64(len(tokens)) - 1, // deduct the AP
				Medium:              medium,
				Source:              "a2p",
				SourceURL:           "https://a2p.bitmark.com/v2/artworks",
				PreviewURL:          fmt.Sprintf("https://d3d03cxftasgml.cloudfront.net/%s", asset.ID),
				ThumbnailURL:        fmt.Sprintf("https://d3d03cxftasgml.cloudfront.net/%s_thumbnail", asset.ID),
				GalleryThumbnailURL: fmt.Sprintf("https://d3d03cxftasgml.cloudfront.net/%s_thumbnail", asset.ID),
				AssetURL:            fmt.Sprintf("https://a2p.bitmark.com/v2/artworks/%s", asset.ID),
			},
			BlockchainMetadata: asset.Metadata,
			Tokens:             tokens,
		}

		if err := db.IndexAsset(ctx, assetID, assetUpdates); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	config.LoadConfig("NFT_INDEXER")

	bitmarksdk.Init(&bitmarksdk.Config{
		Network: bitmarksdk.Network(viper.GetString("bitmarksdk.network")),
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		APIToken: viper.GetString("bitmarksdk.apikey"),
	})

	ctx := context.Background()

	db, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("mongodb.uri"), "nft_indexer")
	if err != nil {
		panic(err)
	}

	if err := importTraderAliases(ctx, db); err != nil {
		panic(err)
	}

	if err := importV1(ctx, db, a2pV1Assets); err != nil {
		panic(err)
	}

	if err := importV2(ctx, db, a2pV2Assets); err != nil {
		panic(err)
	}
}
